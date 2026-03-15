"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { userApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { MessageSquareText, Plus, Loader2, ChevronRight } from "lucide-react";
import { statusColor, formatRelative } from "@/lib/utils";

export default function TicketsPage() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ title: "", description: "" });
  const [error, setError] = useState("");

  const { data: tickets, isLoading } = useQuery({
    queryKey: ["tickets"],
    queryFn: userApi.getTickets,
    retry: false,
  });

  const createMutation = useMutation({
    mutationFn: userApi.createTicket,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tickets"] });
      setOpen(false);
      setForm({ title: "", description: "" });
    },
    onError: (err: Error) => setError(err.message),
  });

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold flex items-center gap-2">
            <MessageSquareText className="h-5 w-5 text-primary" /> Support
          </h1>
          <p className="text-sm text-muted-foreground">Submit and track support requests.</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm"><Plus className="mr-1 h-4 w-4" /> New</Button>
          </DialogTrigger>
          <DialogContent className="max-w-sm mx-auto">
            <DialogHeader>
              <DialogTitle>New Support Request</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 mt-2">
              {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
              <div className="space-y-2">
                <Label>Title</Label>
                <Input
                  placeholder="Brief summary…"
                  value={form.title}
                  onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <Textarea
                  placeholder="Describe the issue…"
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  rows={4}
                />
              </div>
              <Button
                className="w-full"
                onClick={() => { setError(""); createMutation.mutate(form); }}
                disabled={createMutation.isPending || !form.title}
              >
                {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Submit Request
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-20 rounded-lg" />)}
        </div>
      ) : tickets && tickets.length > 0 ? (
        <div className="space-y-2">
          {tickets.map((ticket) => (
            <Link key={ticket.id} href={`/dashboard/tickets/${ticket.id}`}>
              <Card className="hover:bg-accent/50 transition-colors cursor-pointer">
                <CardContent className="flex items-center gap-3 py-3 px-4">
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-sm truncate">{ticket.title}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <Badge variant={statusColor(ticket.status)} className="text-xs capitalize">
                        {ticket.status}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {ticket.replies.length} replies · {formatRelative(ticket.updated_at)}
                      </span>
                    </div>
                  </div>
                  <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-muted-foreground">
          <MessageSquareText className="h-10 w-10 mx-auto mb-3" />
          <p className="text-sm">No support requests yet. Create one if you need help.</p>
        </div>
      )}
    </div>
  );
}
