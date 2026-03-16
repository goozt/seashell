"use client";

import { useState, useEffect, useRef } from "react";
import { useParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { userApi } from "@/lib/api";
import { wsEvents } from "@/hooks/useWebSocket";
import { useAuthStore } from "@/store/auth";
import type { Ticket } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { statusColor, formatDate } from "@/lib/utils";
import { ArrowLeft, Send, Loader2, Info } from "lucide-react";
import { cn } from "@/lib/utils";

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { user } = useAuthStore();
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  const { data: ticket, isLoading } = useQuery({
    queryKey: ["ticket", id],
    queryFn: () => userApi.getTicket(id),
  });

  const replyMutation = useMutation({
    mutationFn: (msg: string) => userApi.replyTicket(id, { message: msg }),
    onSuccess: (updated) => {
      qc.setQueryData(["ticket", id], updated);
      setMessage("");
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }), 100);
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

  // Scroll to bottom when replies load
  useEffect(() => {
    if (ticket?.replies?.length) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [ticket?.replies?.length]);

  if (isLoading)
    return (
      <div className="space-y-3">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24 rounded-xl" />
        <Skeleton className="h-16 rounded-xl" />
      </div>
    );

  if (!ticket)
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">Support request not found.</p>
        <Button asChild variant="outline" className="mt-4">
          <Link href="/dashboard/tickets">← Back</Link>
        </Button>
      </div>
    );

  return (
    <div className="flex flex-col h-[calc(100dvh-8rem)]">
      {/* Header */}
      <div className="flex items-center gap-3 pb-4 border-b border-primary/10 shrink-0">
        <Button asChild variant="ghost" size="icon" className="h-9 w-9 rounded-full hover:bg-primary/10 text-primary">
          <Link href="/dashboard/tickets"><ArrowLeft className="h-4 w-4" /></Link>
        </Button>
        <div className="min-w-0 flex-1">
          <h1 className="text-base font-bold truncate">{ticket.title}</h1>
          <div className="flex items-center gap-2 mt-0.5">
            <Badge variant={statusColor(ticket.status)} className="text-[10px] capitalize">
              {ticket.status}
            </Badge>
            <span className="text-[10px] text-muted-foreground">{formatDate(ticket.created_at)}</span>
          </div>
        </div>
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground shrink-0">
          <Info className="h-4 w-4" />
        </Button>
      </div>

      {/* Messages — scrollable */}
      <div className="flex-1 overflow-y-auto py-4 space-y-4">
        {/* Description as first "message" */}
        {ticket.description && (
          <div className="flex items-start gap-3 max-w-[85%]">
            <div className="size-8 rounded-full bg-primary/20 flex items-center justify-center shrink-0 border border-primary/30 text-xs font-bold text-primary">
              S
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-[11px] font-medium text-muted-foreground ml-1">System</span>
              <div className="glass-admin px-4 py-3 rounded-2xl rounded-tl-none">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">{ticket.description}</p>
              </div>
            </div>
          </div>
        )}

        {ticket.replies.map((reply, idx) => {
          const isMe = reply.author_username === user?.username;
          const initials = reply.author_username?.[0]?.toUpperCase() ?? "?";

          return (
            <div
              key={idx}
              className={cn(
                "flex items-start gap-3 max-w-[85%]",
                isMe && "flex-row-reverse ml-auto"
              )}
            >
              <div
                className={cn(
                  "size-8 rounded-full flex items-center justify-center shrink-0 text-xs font-bold",
                  isMe
                    ? "bg-primary text-primary-foreground shadow-[0_0_12px_rgba(13,223,242,0.3)]"
                    : "bg-primary/20 text-primary border border-primary/30"
                )}
              >
                {initials}
              </div>
              <div className={cn("flex flex-col gap-1", isMe && "items-end")}>
                <span className={cn("text-[11px] font-medium ml-1", isMe ? "text-primary mr-1" : "text-muted-foreground")}>
                  {isMe ? "You" : reply.author_username}
                </span>
                <div className={cn("px-4 py-3 rounded-2xl", isMe ? "glass-member rounded-tr-none" : "glass-admin rounded-tl-none")}>
                  <p className="text-sm leading-relaxed whitespace-pre-wrap">{reply.message}</p>
                </div>
                <span className={cn("text-[10px] text-muted-foreground", isMe ? "mr-1" : "ml-1")}>
                  {formatDate(reply.created_at)}
                </span>
              </div>
            </div>
          );
        })}

        <div ref={bottomRef} />
      </div>

      {/* Reply input */}
      {ticket.status !== "closed" && (
        <div className="shrink-0 pt-3 border-t border-primary/10">
          {error && (
            <Alert variant="destructive" className="mb-2">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <div className="flex items-center gap-2">
            <input
              className="flex-1 bg-primary/5 border border-primary/20 rounded-full py-3 px-5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary placeholder:text-muted-foreground transition-all"
              placeholder="Type your message…"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey && message.trim()) {
                  e.preventDefault();
                  setError("");
                  replyMutation.mutate(message);
                }
              }}
            />
            <Button
              size="icon"
              className="h-11 w-11 rounded-full bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_0_16px_rgba(13,223,242,0.3)] shrink-0"
              onClick={() => { setError(""); replyMutation.mutate(message); }}
              disabled={replyMutation.isPending || !message.trim()}
            >
              {replyMutation.isPending
                ? <Loader2 className="h-4 w-4 animate-spin" />
                : <Send className="h-4 w-4" />}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
