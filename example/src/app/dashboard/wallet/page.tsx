"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userApi, kycApi, verificationApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Wallet, Copy, Check, Plus, RefreshCw, ShieldCheck, ShieldX, ShieldOff, Clock } from "lucide-react";
import Link from "next/link";
import type { FieldDefinition, VerifySubmissionStatus } from "@/types/api";

function StatusBadge({ status }: { status: VerifySubmissionStatus }) {
  const variants: Record<VerifySubmissionStatus, { label: string; variant: "default" | "secondary" | "destructive" }> = {
    approved: { label: "Verified", variant: "default" },
    pending: { label: "Pending Review", variant: "secondary" },
    rejected: { label: "Rejected", variant: "destructive" },
  };
  const { label, variant } = variants[status] ?? variants.pending;
  return <Badge variant={variant}>{label}</Badge>;
}

function StatusIcon({ status }: { status: VerifySubmissionStatus | null }) {
  if (status === "approved") return <ShieldCheck className="h-10 w-10 text-green-500" />;
  if (status === "rejected") return <ShieldX className="h-10 w-10 text-destructive" />;
  if (status === "pending") return <Clock className="h-10 w-10 text-yellow-500" />;
  return <ShieldOff className="h-10 w-10 text-muted-foreground" />;
}

function VerificationForm({
  fields,
  onSubmit,
  isPending,
  error,
}: {
  fields: FieldDefinition[];
  onSubmit: (values: Record<string, string>) => void;
  isPending: boolean;
  error: Error | null;
}) {
  const [values, setValues] = useState<Record<string, string>>({});
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  function handleChange(name: string, value: string) {
    setValues((prev) => ({ ...prev, [name]: value }));
    setValidationErrors((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
  }

  function validate(): boolean {
    const errors: Record<string, string> = {};
    for (const field of fields) {
      const val = values[field.name] ?? "";
      if (field.required && !val.trim()) {
        errors[field.name] = `${field.label} is required`;
        continue;
      }
      if (!val) continue;
      if (field.min_length && val.length < field.min_length) {
        errors[field.name] = `Must be at least ${field.min_length} characters`;
      }
      if (field.max_length && val.length > field.max_length) {
        errors[field.name] = `Must be at most ${field.max_length} characters`;
      }
      if (field.pattern) {
        try {
          if (!new RegExp(field.pattern).test(val)) {
            errors[field.name] = `Does not match the required format`;
          }
        } catch {
          // Invalid regex pattern from server, skip client validation
        }
      }
    }
    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  }

  function handleSubmit() {
    if (validate()) {
      onSubmit(values);
    }
  }

  return (
    <div className="space-y-4">
      {fields.map((field) => (
        <div key={field.name} className="space-y-2">
          <Label htmlFor={field.name}>
            {field.label}
            {field.required && <span className="text-destructive"> *</span>}
          </Label>

          {field.input_type === "textarea" ? (
            <Textarea
              id={field.name}
              placeholder={field.placeholder}
              value={values[field.name] ?? ""}
              onChange={(e) => handleChange(field.name, e.target.value)}
              maxLength={field.max_length || undefined}
            />
          ) : field.input_type === "select" ? (
            <Select
              value={values[field.name] ?? ""}
              onValueChange={(v) => handleChange(field.name, v)}
            >
              <SelectTrigger>
                <SelectValue placeholder={field.placeholder || "Select..."} />
              </SelectTrigger>
              <SelectContent>
                {field.options?.map((opt) => (
                  <SelectItem key={opt} value={opt}>{opt}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id={field.name}
              type={field.input_type === "number" ? "text" : field.input_type}
              inputMode={field.input_type === "number" ? "numeric" : undefined}
              placeholder={field.placeholder}
              value={values[field.name] ?? ""}
              onChange={(e) => {
                let val = e.target.value;
                if (field.input_type === "number") {
                  val = val.replace(/\D/g, "");
                }
                if (field.max_length) {
                  val = val.slice(0, field.max_length);
                }
                handleChange(field.name, val);
              }}
              maxLength={field.max_length || undefined}
            />
          )}

          {field.min_length || field.max_length ? (
            <p className="text-xs text-muted-foreground">
              {field.min_length && field.max_length && field.min_length === field.max_length
                ? `Exactly ${field.min_length} characters`
                : field.min_length && field.max_length
                ? `${field.min_length}–${field.max_length} characters`
                : field.min_length
                ? `At least ${field.min_length} characters`
                : `At most ${field.max_length} characters`}
            </p>
          ) : null}

          {validationErrors[field.name] && (
            <p className="text-xs text-destructive">{validationErrors[field.name]}</p>
          )}
        </div>
      ))}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{(error as Error).message}</AlertDescription>
        </Alert>
      )}

      <Button onClick={handleSubmit} disabled={isPending} className="w-full">
        {isPending ? "Submitting…" : "Submit Verification"}
      </Button>
    </div>
  );
}

export default function WalletPage() {
  const qc = useQueryClient();
  const [copied, setCopied] = useState(false);
  const { user } = useAuthStore();
  const isAffiliated = !!user?.authority_id;

  const { data: kyc, isLoading: kycLoading } = useQuery({
    queryKey: ["kyc"],
    queryFn: kycApi.getMyKYC,
    retry: false,
    enabled: !isAffiliated,
  });

  const { data: verification, isLoading: verifyLoading } = useQuery({
    queryKey: ["verification"],
    queryFn: verificationApi.getMyVerification,
    retry: false,
    enabled: isAffiliated,
  });

  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ["wallet"],
    queryFn: userApi.getWallet,
    retry: false,
  });

  const createMutation = useMutation({
    mutationFn: userApi.createWallet,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["wallet"] }),
  });

  const submitMutation = useMutation({
    mutationFn: (fieldValues: Record<string, string>) =>
      verificationApi.submitVerification({ field_values: fieldValues }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["verification"] }),
  });

  const verifyCheckLoading = isAffiliated ? verifyLoading : kycLoading;
  const isVerified = isAffiliated
    ? verification?.submission?.status === "approved"
    : kyc?.status === "verified";

  // Auto-create wallet once verification is approved
  useEffect(() => {
    if (isVerified && !wallet && !walletLoading && !createMutation.isPending && !createMutation.isSuccess) {
      createMutation.mutate();
    }
  }, [isVerified, wallet, walletLoading]);

  function copyAddress() {
    if (!wallet?.address) return;
    navigator.clipboard.writeText(wallet.address);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  // --- Affiliated user: show verification flow until verified ---
  if (isAffiliated && !verifyLoading && !isVerified) {
    const notConfigured = !verification || verification.config_status === "not_configured";
    const submission = verification?.submission;
    const config = verification?.config;

    return (
      <div className="flex flex-col min-h-[calc(100dvh-12rem)]">
        <div className="mb-5">
          <h1 className="text-xl font-bold flex items-center gap-2">
            <Wallet className="h-5 w-5 text-primary" /> Wallet
          </h1>
          <p className="text-sm text-muted-foreground">
            Complete verification to set up your wallet.
          </p>
        </div>

        {notConfigured && (
          <Card className="flex-1 flex flex-col">
            <CardContent className="flex-1 flex flex-col items-center justify-center text-center py-8">
              <Clock className="h-10 w-10 text-muted-foreground mb-3" />
              <p className="font-medium">Authority setup pending</p>
              <p className="text-sm text-muted-foreground mt-1">
                Your authority has not configured the verification process yet. Please try again later.
              </p>
            </CardContent>
          </Card>
        )}

        {submission?.status === "pending" && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Verification Status</CardTitle>
                <div className="flex items-center gap-2">
                  <StatusBadge status="pending" />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => qc.invalidateQueries({ queryKey: ["verification"] })}
                    title="Refresh status"
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-4">
                <StatusIcon status="pending" />
                <div>
                  <p className="font-semibold">Under review</p>
                  <p className="text-sm text-muted-foreground">
                    Your verification has been submitted and is awaiting review by the authority owner.
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Submitted on {new Date(submission.created_at).toLocaleDateString()}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {submission?.status === "rejected" && config && (
          <>
            <Alert variant="destructive">
              <ShieldX className="h-4 w-4" />
              <AlertDescription>
                <span className="font-medium">Verification rejected.</span>
                {submission.remarks && (
                  <span className="block mt-1">{submission.remarks}</span>
                )}
              </AlertDescription>
            </Alert>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Resubmit Verification</CardTitle>
                <CardDescription>
                  Please correct the issues noted above and submit again.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <VerificationForm
                  fields={config.fields}
                  onSubmit={(values) => submitMutation.mutate(values)}
                  isPending={submitMutation.isPending}
                  error={submitMutation.error}
                />
              </CardContent>
            </Card>
          </>
        )}

        {!notConfigured && !submission && config && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Verify your identity</CardTitle>
              <CardDescription>
                Fill out the fields below as required by your authority.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <VerificationForm
                fields={config.fields}
                onSubmit={(values) => submitMutation.mutate(values)}
                isPending={submitMutation.isPending}
                error={submitMutation.error}
              />
            </CardContent>
          </Card>
        )}
      </div>
    );
  }

  // --- Non-affiliated user: KYC required ---
  if (!isAffiliated && !kycLoading && !isVerified) {
    return (
      <div className="space-y-5">
        <div>
          <h1 className="text-xl font-bold flex items-center gap-2">
            <Wallet className="h-5 w-5 text-primary" /> Wallet
          </h1>
          <p className="text-sm text-muted-foreground">Manage your SHELL balance and address.</p>
        </div>
        <Alert>
          <ShieldOff className="h-4 w-4" />
          <AlertDescription>
            KYC verification required before creating a wallet.{" "}
            <Link href="/dashboard/kyc" className="underline font-medium">Verify now →</Link>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // --- Verified: wallet loading / creating / display ---
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Wallet className="h-5 w-5 text-primary" /> Wallet
        </h1>
        <p className="text-sm text-muted-foreground">Manage your SHELL balance and address.</p>
      </div>

      {verifyCheckLoading || walletLoading ? (
        <Card>
          <CardContent className="py-6 space-y-3">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-full" />
          </CardContent>
        </Card>
      ) : wallet ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Balance</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-4xl font-bold">
                {wallet.balance}
                <span className="text-lg font-normal text-muted-foreground ml-2">SHELL</span>
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Address</CardTitle>
              <CardDescription>Your public Base58 wallet address</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-2 p-3 rounded-md bg-muted">
                <p className="font-mono text-xs break-all flex-1">{wallet.address}</p>
                <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={copyAddress}>
                  {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => qc.invalidateQueries({ queryKey: ["wallet"] })}
                className="w-full"
              >
                <RefreshCw className="mr-2 h-4 w-4" /> Refresh Balance
              </Button>
            </CardContent>
          </Card>
        </>
      ) : (
        <Card>
          <CardContent className="py-8 text-center space-y-4">
            <Wallet className="h-12 w-12 mx-auto text-muted-foreground" />
            <div>
              <p className="font-semibold">Setting up your wallet…</p>
              <p className="text-sm text-muted-foreground mt-1">
                {createMutation.isPending ? "Creating wallet, please wait." : "Your wallet will be ready shortly."}
              </p>
            </div>
            {createMutation.error && (
              <>
                <Alert variant="destructive">
                  <AlertDescription>{(createMutation.error as Error).message}</AlertDescription>
                </Alert>
                <Button onClick={() => createMutation.mutate()} disabled={createMutation.isPending}>
                  <Plus className="mr-2 h-4 w-4" /> Retry
                </Button>
              </>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
