"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { kycApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { ShieldCheck, ShieldX, Clock, ShieldOff } from "lucide-react";
import type { KYCStatus } from "@/types/api";

function KYCBadge({ status }: { status: KYCStatus }) {
  const variants: Record<KYCStatus, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
    verified: { label: "Verified", variant: "default" },
    pending: { label: "Pending", variant: "secondary" },
    rejected: { label: "Rejected", variant: "destructive" },
    unverified: { label: "Unverified", variant: "outline" },
  };
  const { label, variant } = variants[status] ?? variants.unverified;
  return <Badge variant={variant}>{label}</Badge>;
}

function KYCIcon({ status }: { status: KYCStatus }) {
  if (status === "verified") return <ShieldCheck className="h-10 w-10 text-green-500" />;
  if (status === "rejected") return <ShieldX className="h-10 w-10 text-destructive" />;
  if (status === "pending") return <Clock className="h-10 w-10 text-yellow-500" />;
  return <ShieldOff className="h-10 w-10 text-muted-foreground" />;
}

export default function KYCPage() {
  const qc = useQueryClient();
  const [nationalId, setNationalId] = useState("");
  const [taxId, setTaxId] = useState("");

  const { data: kyc, isLoading } = useQuery({
    queryKey: ["kyc"],
    queryFn: kycApi.getMyKYC,
    retry: false,
  });

  const initiateMutation = useMutation({
    mutationFn: () => kycApi.initiateKYC({ national_id: nationalId, tax_id: taxId || undefined }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["kyc"] });
      setNationalId("");
      setTaxId("");
    },
  });

  const canSubmit = /^\d{12}$/.test(nationalId);

  if (isLoading) {
    return <div className="p-6 text-sm text-muted-foreground">Loading…</div>;
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" /> Identity Verification
        </h1>
        <p className="text-sm text-muted-foreground">
          Verify your identity to enable wallet creation and transactions.
        </p>
      </div>

      {kyc ? (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Verification Status</CardTitle>
              <KYCBadge status={kyc.status} />
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-4">
              <KYCIcon status={kyc.status} />
              <div>
                {kyc.verified_name && (
                  <p className="font-semibold">{kyc.verified_name}</p>
                )}
                {kyc.national_id_masked && (
                  <p className="text-sm text-muted-foreground font-mono">{kyc.national_id_masked}</p>
                )}
                {kyc.verified_at && (
                  <p className="text-xs text-muted-foreground">
                    Verified {new Date(kyc.verified_at).toLocaleDateString()}
                  </p>
                )}
                {kyc.status === "rejected" && kyc.rejected_reason && (
                  <p className="text-sm text-destructive mt-1">{kyc.rejected_reason}</p>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Submit Verification</CardTitle>
            <CardDescription>
              Enter your 12-digit National ID. Your ID is never stored — only a masked version is kept after verification.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="national-id">National ID <span className="text-destructive">*</span></Label>
              <Input
                id="national-id"
                placeholder="123456789012"
                value={nationalId}
                onChange={(e) => setNationalId(e.target.value.replace(/\D/g, "").slice(0, 12))}
                maxLength={12}
              />
              <p className="text-xs text-muted-foreground">12 digits, no spaces or dashes</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="tax-id">Tax ID <span className="text-muted-foreground">(optional)</span></Label>
              <Input
                id="tax-id"
                placeholder="Tax identification number"
                value={taxId}
                onChange={(e) => setTaxId(e.target.value)}
              />
            </div>
            {initiateMutation.error && (
              <Alert variant="destructive">
                <AlertDescription>{(initiateMutation.error as Error).message}</AlertDescription>
              </Alert>
            )}
            <Button
              onClick={() => initiateMutation.mutate()}
              disabled={!canSubmit || initiateMutation.isPending}
              className="w-full"
            >
              {initiateMutation.isPending ? "Verifying…" : "Verify Identity"}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
