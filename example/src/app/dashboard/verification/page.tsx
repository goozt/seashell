"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { verificationApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ShieldCheck, ShieldX, Clock, ShieldOff, AlertCircle } from "lucide-react";
import { useAuthStore } from "@/store/auth";
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

export default function VerificationPage() {
  const qc = useQueryClient();
  const { user } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ["verification"],
    queryFn: verificationApi.getMyVerification,
    retry: false,
  });

  const submitMutation = useMutation({
    mutationFn: (fieldValues: Record<string, string>) =>
      verificationApi.submitVerification({ field_values: fieldValues }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["verification"] });
    },
  });

  if (isLoading) {
    return <div className="p-6 text-sm text-muted-foreground">Loading…</div>;
  }

  const notAffiliated = !user?.authority_id;
  const notConfigured = !data || data.config_status === "not_configured";
  const submission = data?.submission;
  const config = data?.config;

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" /> Verification
        </h1>
        <p className="text-sm text-muted-foreground">
          Complete verification to enable wallet creation and transactions.
        </p>
      </div>

      {/* Not affiliated with any authority */}
      {notAffiliated && (
        <Card>
          <CardContent className="py-8 text-center">
            <AlertCircle className="h-10 w-10 text-muted-foreground mx-auto mb-3" />
            <p className="text-sm text-muted-foreground">
              You must join an authority before verification can begin.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Authority hasn't configured verification */}
      {!notAffiliated && notConfigured && (
        <Card>
          <CardContent className="py-8 text-center">
            <Clock className="h-10 w-10 text-muted-foreground mx-auto mb-3" />
            <p className="font-medium">Authority setup pending</p>
            <p className="text-sm text-muted-foreground mt-1">
              Your authority has not configured the verification process yet. Please try again later.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Submission approved */}
      {submission?.status === "approved" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Verification Status</CardTitle>
              <StatusBadge status="approved" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-4">
              <StatusIcon status="approved" />
              <div>
                <p className="font-semibold">Verification complete</p>
                {submission.reviewed_at && (
                  <p className="text-xs text-muted-foreground">
                    Approved on {new Date(submission.reviewed_at).toLocaleDateString()}
                  </p>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Submission pending */}
      {submission?.status === "pending" && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Verification Status</CardTitle>
              <StatusBadge status="pending" />
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

      {/* Submission rejected -- show remarks + form for resubmission */}
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

      {/* No submission yet -- show form */}
      {!notAffiliated && !notConfigured && !submission && config && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Submit Verification</CardTitle>
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
