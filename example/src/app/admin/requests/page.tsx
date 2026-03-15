"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { statusColor, formatRelative } from "@/lib/utils";
import { FileQuestion, CheckCircle, XCircle, Loader2 } from "lucide-react";
import type { Authority } from "@/types/api";

export default function AuthorityRequestsPage() {
  const qc = useQueryClient();
  const [selectedReq, setSelectedReq] = useState<Authority | null>(null);
  const [action, setAction] = useState<"approve" | "reject" | null>(null);
  const [approveForm, setApproveForm] = useState({ base_price: "1.0", sensitivity_k: "0.1" });
  const [rejectReason, setRejectReason] = useState("");
  const [error, setError] = useState("");

  const { data: requests, isLoading } = useQuery({
    queryKey: ["authority-requests"],
    queryFn: adminApi.getRequests,
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, base_price, sensitivity_k }: { id: string; base_price: number; sensitivity_k: number }) =>
      adminApi.approveRequest(id, { base_price, sensitivity_k }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["authority-requests"] });
      setSelectedReq(null);
      setAction(null);
    },
    onError: (err: Error) => setError(err.message),
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      adminApi.rejectRequest(id, { reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["authority-requests"] });
      setSelectedReq(null);
      setAction(null);
    },
    onError: (err: Error) => setError(err.message),
  });

  const pending = requests?.filter((r) => r.status === "pending") ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <FileQuestion className="h-5 w-5 text-primary" /> Authority Requests
        </h1>
        <p className="text-sm text-muted-foreground">{pending.length} pending review.</p>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {[1, 2].map((i) => <Skeleton key={i} className="h-24 rounded-lg" />)}
        </div>
      ) : (requests?.length ?? 0) === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-8">No authority requests.</p>
      ) : (
        <div className="space-y-2">
          {requests?.map((req) => (
            <Card key={req.id}>
              <CardContent className="py-3 px-4">
                <div className="flex items-start justify-between">
                  <div className="min-w-0">
                    <p className="font-semibold text-sm">{req.name}</p>
                    {req.description && (
                      <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{req.description}</p>
                    )}
                    <p className="text-xs text-muted-foreground mt-1">{formatRelative(req.created_at)}</p>
                  </div>
                  <Badge variant={statusColor(req.status)} className="ml-2 shrink-0 capitalize">{req.status}</Badge>
                </div>
                {req.status === "pending" && (
                  <div className="flex gap-2 mt-3">
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1 text-green-600 border-green-600 hover:bg-green-50"
                      onClick={() => { setSelectedReq(req); setAction("approve"); setError(""); }}
                    >
                      <CheckCircle className="mr-1 h-3.5 w-3.5" /> Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1 text-destructive border-destructive hover:bg-red-50"
                      onClick={() => { setSelectedReq(req); setAction("reject"); setError(""); }}
                    >
                      <XCircle className="mr-1 h-3.5 w-3.5" /> Reject
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Approve Dialog */}
      <Dialog open={action === "approve" && !!selectedReq} onOpenChange={() => { setAction(null); setSelectedReq(null); }}>
        <DialogContent className="max-w-sm mx-auto">
          <DialogHeader>
            <DialogTitle>Approve "{selectedReq?.name}"</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 mt-2">
            {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
            <div className="space-y-2">
              <Label>Base Price (SHELL)</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={approveForm.base_price}
                onChange={(e) => setApproveForm((f) => ({ ...f, base_price: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label>Sensitivity K</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={approveForm.sensitivity_k}
                onChange={(e) => setApproveForm((f) => ({ ...f, sensitivity_k: e.target.value }))}
              />
            </div>
            <Button
              className="w-full"
              onClick={() => {
                if (!selectedReq) return;
                approveMutation.mutate({
                  id: selectedReq.id,
                  base_price: parseFloat(approveForm.base_price),
                  sensitivity_k: parseFloat(approveForm.sensitivity_k),
                });
              }}
              disabled={approveMutation.isPending}
            >
              {approveMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Approve Authority
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Reject Dialog */}
      <Dialog open={action === "reject" && !!selectedReq} onOpenChange={() => { setAction(null); setSelectedReq(null); }}>
        <DialogContent className="max-w-sm mx-auto">
          <DialogHeader>
            <DialogTitle>Reject "{selectedReq?.name}"</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 mt-2">
            {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
            <div className="space-y-2">
              <Label>Reason (optional)</Label>
              <Input
                placeholder="Reason for rejection…"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
              />
            </div>
            <Button
              variant="destructive"
              className="w-full"
              onClick={() => {
                if (!selectedReq) return;
                rejectMutation.mutate({ id: selectedReq.id, reason: rejectReason || undefined });
              }}
              disabled={rejectMutation.isPending}
            >
              {rejectMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Confirm Rejection
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
