"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { userApi } from "@/lib/api";
import { wsEvents } from "@/hooks/useWebSocket";
import type { Ticket } from "@/types/api";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { statusColor, formatDate } from "@/lib/utils";
import { ArrowLeft, Send, Loader2 } from "lucide-react";

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const { data: ticket, isLoading } = useQuery({
    queryKey: ["ticket", id],
    queryFn: () => userApi.getTicket(id),
  });

  const replyMutation = useMutation({
    mutationFn: (msg: string) => userApi.replyTicket(id, { message: msg }),
    onSuccess: (updated) => {
      qc.setQueryData(["ticket", id], updated);
      setMessage("");
    },
    onError: (err: Error) => setError(err.message),
  });

  useEffect(() => {
    if (!ticket) return;
    const unsub = wsEvents.onSupport(ticket.id, (_ticketId, reply) => {
      qc.setQueryData(["ticket", id], (old: Ticket | undefined) =>
        old ? { ...old, replies: [...old.replies, reply] } : old
      );
    });
    return unsub;
  }, [ticket?.id, qc, id]);

  if (isLoading) return (
    <div className="space-y-3">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-24 rounded-lg" />
      <Skeleton className="h-16 rounded-lg" />
    </div>
  );

  if (!ticket) return (
    <div className="text-center py-12">
      <p className="text-muted-foreground">Support request not found.</p>
      <Button asChild variant="outline" className="mt-4">
        <Link href="/dashboard/tickets">← Back</Link>
      </Button>
    </div>
  );

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-3">
        <Button asChild variant="ghost" size="icon" className="h-8 w-8">
          <Link href="/dashboard/tickets"><ArrowLeft className="h-4 w-4" /></Link>
        </Button>
        <div className="min-w-0">
          <h1 className="text-lg font-bold truncate">{ticket.title}</h1>
          <div className="flex items-center gap-2 mt-0.5">
            <Badge variant={statusColor(ticket.status)} className="text-xs capitalize">{ticket.status}</Badge>
            <span className="text-xs text-muted-foreground">{formatDate(ticket.created_at)}</span>
          </div>
        </div>
      </div>

      {ticket.description && (
        <Card>
          <CardContent className="py-3 px-4">
            <p className="text-sm whitespace-pre-wrap">{ticket.description}</p>
          </CardContent>
        </Card>
      )}

      {/* Replies */}
      <div className="space-y-3">
        <h2 className="text-sm font-semibold">
          {ticket.replies.length} {ticket.replies.length === 1 ? "Reply" : "Replies"}
        </h2>
        {ticket.replies.map((reply, idx) => (
          <Card key={idx}>
            <CardContent className="py-3 px-4">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs font-semibold">{reply.author_username}</span>
                <span className="text-xs text-muted-foreground">{formatDate(reply.created_at)}</span>
              </div>
              <p className="text-sm whitespace-pre-wrap">{reply.message}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Reply form */}
      {ticket.status !== "closed" && (
        <div className="space-y-3">
          <h2 className="text-sm font-semibold">Add Reply</h2>
          {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
          <Textarea
            placeholder="Your message…"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={4}
          />
          <Button
            className="w-full"
            onClick={() => { setError(""); replyMutation.mutate(message); }}
            disabled={replyMutation.isPending || !message.trim()}
          >
            {replyMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Send className="mr-2 h-4 w-4" />}
            Send Reply
          </Button>
        </div>
      )}
    </div>
  );
}
