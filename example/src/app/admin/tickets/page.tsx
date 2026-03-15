"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { wsEvents } from "@/hooks/useWebSocket";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { statusColor, formatRelative, formatDate } from "@/lib/utils";
import { TicketCheck, ChevronRight, Send, Loader2, ArrowUpCircle } from "lucide-react";
import type { Ticket, TicketLevel } from "@/types/api";

const STATUSES = ["open", "in_progress", "resolved", "closed"] as const;

type Tab = "node" | "regional";

function EscalationBadge({ level }: { level?: TicketLevel }) {
  if (!level || level === "node") return null;
  return (
    <Badge variant="secondary" className="text-xs">
      {level === "regional" ? "↑ Regional" : "↑↑ Superadmin"}
    </Badge>
  );
}

function TicketList({
  tickets,
  isLoading,
  emptyText,
  onSelect,
}: {
  tickets: Ticket[];
  isLoading: boolean;
  emptyText: string;
  onSelect: (t: Ticket) => void;
}) {
  if (isLoading) return <div className="space-y-2">{[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 rounded-lg" />)}</div>;
  if (tickets.length === 0) return <p className="text-sm text-muted-foreground text-center py-8">{emptyText}</p>;
  return (
    <div className="space-y-2">
      {tickets.map((ticket) => (
        <Card key={ticket.id} className="cursor-pointer hover:bg-accent/50 transition-colors" onClick={() => onSelect(ticket)}>
          <CardContent className="flex items-center gap-3 py-3 px-4">
            <div className="flex-1 min-w-0">
              <p className="font-medium text-sm truncate">{ticket.title}</p>
              <div className="flex items-center gap-2 mt-1 flex-wrap">
                <Badge variant={statusColor(ticket.status)} className="text-xs capitalize">{ticket.status}</Badge>
                <EscalationBadge level={ticket.escalated_to} />
                <span className="text-xs text-muted-foreground">{ticket.replies.length} replies · {formatRelative(ticket.updated_at)}</span>
              </div>
            </div>
            <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export default function AdminTicketsPage() {
  const qc = useQueryClient();
  const [selected, setSelected] = useState<Ticket | null>(null);
  const [reply, setReply] = useState("");
  const [escalateNote, setEscalateNote] = useState("");
  const [showEscalate, setShowEscalate] = useState(false);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<Tab>("node");

  const { data: nodeTickets, isLoading: nodeLoading } = useQuery({
    queryKey: ["admin-tickets", "node"],
    queryFn: adminApi.getTickets,
  });

  const { data: regionalTickets, isLoading: regionalLoading } = useQuery({
    queryKey: ["admin-tickets", "regional"],
    queryFn: adminApi.getRegionalTickets,
  });

  function invalidate() {
    qc.invalidateQueries({ queryKey: ["admin-tickets"] });
  }

  const replyMutation = useMutation({
    mutationFn: (msg: string) => adminApi.replyTicket(selected!.id, { message: msg }),
    onSuccess: (updated) => { invalidate(); setSelected(updated); setReply(""); },
    onError: (err: Error) => setError(err.message),
  });

  const statusMutation = useMutation({
    mutationFn: (status: string) => adminApi.updateTicket(selected!.id, { status }),
    onSuccess: (updated) => { invalidate(); setSelected(updated); },
    onError: (err: Error) => setError(err.message),
  });

  const escalateRegionalMutation = useMutation({
    mutationFn: () => adminApi.escalateToRegional(selected!.id, escalateNote),
    onSuccess: (updated) => { invalidate(); setSelected(null); setShowEscalate(false); setEscalateNote(""); },
    onError: (err: Error) => setError(err.message),
  });

  const escalateSuperMutation = useMutation({
    mutationFn: () => adminApi.escalateToSuper(selected!.id, escalateNote),
    onSuccess: (updated) => { invalidate(); setSelected(null); setShowEscalate(false); setEscalateNote(""); },
    onError: (err: Error) => setError(err.message),
  });

  const isRegional = selected?.escalated_to === "regional";

  useEffect(() => {
    if (!selected) return;
    const unsub = wsEvents.onSupport(selected.id, (_ticketId, reply) => {
      setSelected((prev) =>
        prev ? { ...prev, replies: [...prev.replies, reply] } : prev
      );
      invalidate();
    });
    return unsub;
  }, [selected?.id]);

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <TicketCheck className="h-5 w-5 text-primary" /> Support
        </h1>
        <p className="text-sm text-muted-foreground">Manage support requests and escalations.</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b">
        {(["node", "regional"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`pb-2 px-1 text-sm font-medium border-b-2 transition-colors ${
              tab === t ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {t === "node" ? "Node Level" : "Regional Level"}
            {t === "regional" && (regionalTickets?.length ?? 0) > 0 && (
              <span className="ml-1.5 bg-orange-100 text-orange-700 text-xs px-1.5 py-0.5 rounded-full">
                {regionalTickets!.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {tab === "node" ? (
        <TicketList
          tickets={nodeTickets ?? []}
          isLoading={nodeLoading}
          emptyText="No support requests at node level."
          onSelect={(t) => { setSelected(t); setError(""); setReply(""); setShowEscalate(false); }}
        />
      ) : (
        <TicketList
          tickets={regionalTickets ?? []}
          isLoading={regionalLoading}
          emptyText="No support requests escalated to regional level."
          onSelect={(t) => { setSelected(t); setError(""); setReply(""); setShowEscalate(false); }}
        />
      )}

      {/* Ticket Detail Dialog */}
      <Dialog open={!!selected} onOpenChange={(open) => { if (!open) setSelected(null); }}>
        <DialogContent className="max-w-lg mx-auto max-h-[90vh] overflow-y-auto">
          {selected && (
            <>
              <DialogHeader>
                <DialogTitle className="text-base pr-6">{selected.title}</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 mt-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <EscalationBadge level={selected.escalated_to} />
                  {selected.escalation_note && (
                    <span className="text-xs text-muted-foreground italic">"{selected.escalation_note}"</span>
                  )}
                </div>

                {/* Status selector */}
                <div className="flex flex-wrap gap-1.5">
                  {STATUSES.map((s) => (
                    <Button
                      key={s}
                      size="sm"
                      variant={selected.status === s ? "default" : "outline"}
                      className="text-xs capitalize h-7"
                      onClick={() => statusMutation.mutate(s)}
                      disabled={statusMutation.isPending}
                    >
                      {s.replace("_", " ")}
                    </Button>
                  ))}
                </div>

                {selected.description && (
                  <p className="text-sm border p-3 rounded-md whitespace-pre-wrap">{selected.description}</p>
                )}

                {/* Replies */}
                <div className="space-y-2">
                  {selected.replies.map((r, i) => (
                    <div key={i} className="border rounded-md p-3">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-semibold">{r.author_username}</span>
                        <Badge variant="outline" className="text-xs capitalize">{r.author_role}</Badge>
                        <span className="text-xs text-muted-foreground">{formatDate(r.created_at)}</span>
                      </div>
                      <p className="text-sm whitespace-pre-wrap">{r.message}</p>
                    </div>
                  ))}
                </div>

                {/* Reply form */}
                {selected.status !== "closed" && (
                  <div className="space-y-2">
                    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
                    <Textarea
                      placeholder="Admin reply…"
                      value={reply}
                      onChange={(e) => setReply(e.target.value)}
                      rows={3}
                    />
                    <Button
                      className="w-full"
                      size="sm"
                      onClick={() => { setError(""); replyMutation.mutate(reply); }}
                      disabled={replyMutation.isPending || !reply.trim()}
                    >
                      {replyMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Send className="mr-2 h-4 w-4" />}
                      Send Reply
                    </Button>
                  </div>
                )}

                {/* Escalation */}
                {selected.status !== "closed" && selected.status !== "resolved" && (
                  <div className="border-t pt-3 space-y-2">
                    {!showEscalate ? (
                      <Button
                        variant="outline"
                        size="sm"
                        className="w-full text-orange-600 border-orange-200 hover:bg-orange-50"
                        onClick={() => setShowEscalate(true)}
                      >
                        <ArrowUpCircle className="mr-2 h-4 w-4" />
                        {isRegional ? "Escalate to Superadmin" : "Escalate to Regional"}
                      </Button>
                    ) : (
                      <div className="space-y-2">
                        <p className="text-xs font-medium text-muted-foreground">
                          {isRegional ? "Escalate to superadmin" : "Escalate to regional admin"}
                        </p>
                        <Input
                          placeholder="Reason (optional)"
                          value={escalateNote}
                          onChange={(e) => setEscalateNote(e.target.value)}
                        />
                        <div className="flex gap-2">
                          <Button variant="outline" size="sm" className="flex-1" onClick={() => { setShowEscalate(false); setEscalateNote(""); }}>
                            Cancel
                          </Button>
                          <Button
                            size="sm"
                            className="flex-1 bg-orange-600 hover:bg-orange-700"
                            disabled={escalateRegionalMutation.isPending || escalateSuperMutation.isPending}
                            onClick={() => {
                              setError("");
                              if (isRegional) escalateSuperMutation.mutate();
                              else escalateRegionalMutation.mutate();
                            }}
                          >
                            {(escalateRegionalMutation.isPending || escalateSuperMutation.isPending)
                              ? <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              : <ArrowUpCircle className="mr-2 h-4 w-4" />}
                            Confirm Escalation
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
