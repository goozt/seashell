"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { kycApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ShieldCheck } from "lucide-react";
import type { KYCRecord, KYCStatus } from "@/types/api";

function KYCBadge({ status }: { status: KYCStatus }) {
  const map: Record<KYCStatus, "default" | "secondary" | "destructive" | "outline"> = {
    verified: "default", pending: "secondary", rejected: "destructive", unverified: "outline",
  };
  return <Badge variant={map[status] ?? "outline"}>{status}</Badge>;
}

export default function AdminKYCPage() {
  const qc = useQueryClient();
  const [rejectTarget, setRejectTarget] = useState<KYCRecord | null>(null);
  const [reason, setReason] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-kyc"],
    queryFn: () => kycApi.listKYC(),
  });

  const { data: summary } = useQuery({
    queryKey: ["admin-kyc-summary"],
    queryFn: kycApi.getSummary,
  });

  const rejectMutation = useMutation({
    mutationFn: ({ userID, reason }: { userID: string; reason: string }) =>
      kycApi.rejectKYC(userID, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-kyc"] });
      qc.invalidateQueries({ queryKey: ["admin-kyc-summary"] });
      setRejectTarget(null);
      setReason("");
    },
  });

  const records = data?.records ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" /> KYC Management
        </h1>
        <p className="text-sm text-muted-foreground">Review identity verification records.</p>
      </div>

      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {[
            { label: "Total", value: summary.total },
            { label: "Verified", value: summary.verified },
            { label: "Pending", value: summary.pending },
            { label: "Rejected", value: summary.rejected },
            { label: "Unverified", value: summary.unverified },
          ].map(({ label, value }) => (
            <Card key={label}>
              <CardContent className="py-3 text-center">
                <p className="text-2xl font-bold">{value}</p>
                <p className="text-xs text-muted-foreground">{label}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Records</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : records.length === 0 ? (
            <p className="text-sm text-muted-foreground">No records found.</p>
          ) : (
            <div className="divide-y">
              {records.map((rec) => (
                <div key={rec.user_id} className="py-3 flex items-center justify-between gap-4">
                  <div>
                    <p className="text-sm font-mono">{rec.national_id_masked}</p>
                    {rec.verified_name && (
                      <p className="text-xs text-muted-foreground">{rec.verified_name}</p>
                    )}
                    <p className="text-xs text-muted-foreground">User: {rec.user_id}</p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <KYCBadge status={rec.status} />
                    {rec.status !== "rejected" && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setRejectTarget(rec)}
                      >
                        Reject
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={!!rejectTarget} onOpenChange={(open) => { if (!open) { setRejectTarget(null); setReason(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject KYC</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Label>Reason</Label>
            <Input
              placeholder="Reason for rejection"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            {rejectMutation.error && (
              <Alert variant="destructive">
                <AlertDescription>{(rejectMutation.error as Error).message}</AlertDescription>
              </Alert>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={() => { setRejectTarget(null); setReason(""); }}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={!reason.trim() || rejectMutation.isPending}
              onClick={() => rejectTarget && rejectMutation.mutate({ userID: rejectTarget.user_id, reason })}
            >
              {rejectMutation.isPending ? "Rejecting…" : "Reject"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
