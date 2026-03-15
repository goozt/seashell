"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { superAdminApi } from "@/lib/api";
import { wsEvents } from "@/hooks/useWebSocket";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { statusColor, formatRelative, formatDate } from "@/lib/utils";
import { TicketCheck, ChevronRight, Send, Loader2 } from "lucide-react";
import type { Ticket } from "@/types/api";

const STATUSES = ["open", "in_progress", "resolved", "closed"] as const;

export default function SuperAdminTicketsPage() {
  const qc = useQueryClient();
  const [selected, setSelected] = useState<Ticket | null>(null);
  const [reply, setReply] = useState("");
  const [error, setError] = useState("");

  const { data: tickets, isLoading } = useQuery({
    queryKey: ["superadmin-tickets"],
    queryFn: superAdminApi.getTickets,
  });

  function invalidate() {
    qc.invalidateQueries({ queryKey: ["superadmin-tickets"] });
  }

  const replyMutation = useMutation({
    mutationFn: (msg: string) => superAdminApi.replyTicket(selected!.id, { message: msg }),
    onSuccess: (updated) => { invalidate(); setSelected(updated); setReply(""); },
    onError: (err: Error) => setError(err.message),
  });

  const statusMutation = useMutation({
    mutationFn: (status: string) => superAdminApi.updateTicket(selected!.id, { status }),
    onSuccess: (updated) => { invalidate(); setSelected(updated); },
    onError: (err: Error) => setError(err.message),
  });

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
          <TicketCheck className="h-5 w-5 text-primary" /> Escalated Support
        </h1>
        <p className="text-sm text-muted-foreground">Support requests escalated to superadmin.</p>
      </div>

      {isLoading ? (
        <div className="space-y-2">{[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 rounded-lg" />)}</div>
      ) : (tickets?.length ?? 0) === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-8">No escalated requests.</p>
      ) : (
        <div className="space-y-2">
          {tickets!.map((ticket) => (
            <Card key={ticket.id} className="cursor-pointer hover:bg-accent/50 transition-colors" onClick={() => { setSelected(ticket); setError(""); setReply(""); }}>
              <CardContent className="flex items-center gap-3 py-3 px-4">
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm truncate">{ticket.title}</p>
                  <div className="flex items-center gap-2 mt-1">
                    <Badge variant={statusColor(ticket.status)} className="text-xs capitalize">{ticket.status}</Badge>
                    <Badge variant="secondary" className="text-xs">↑↑ Escalated</Badge>
                    <span className="text-xs text-muted-foreground">{ticket.replies.length} replies · {formatRelative(ticket.updated_at)}</span>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={!!selected} onOpenChange={(open) => { if (!open) setSelected(null); }}>
        <DialogContent className="max-w-lg mx-auto max-h-[90vh] overflow-y-auto">
          {selected && (
            <>
              <DialogHeader>
                <DialogTitle className="text-base pr-6">{selected.title}</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 mt-2">
                {selected.escalation_note && (
                  <p className="text-xs text-muted-foreground bg-orange-50 border border-orange-200 rounded-md p-2">
                    <span className="font-medium">Escalation note:</span> {selected.escalation_note}
                  </p>
                )}

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
                  <p className="text-sm text-muted-foreground bg-muted p-3 rounded-md whitespace-pre-wrap">{selected.description}</p>
                )}

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

                {selected.status !== "closed" && (
                  <div className="space-y-2">
                    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
                    <Textarea
                      placeholder="Superadmin reply…"
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
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
