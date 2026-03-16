"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userApi, authorityApi, verificationApi, authApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { statusColor, truncateHash, formatRelative } from "@/lib/utils";
import { Building2, Users, Clipboard, Plus, Trash2, Loader2, Copy, Check, ShieldCheck, X, RefreshCw } from "lucide-react";
import type { FieldDefinition, VerificationSubmission } from "@/types/api";

export default function AuthorityPage() {
  const qc = useQueryClient();
  const { hasHydrated, user, setUser } = useAuthStore();
  const isOwner = user?.authority_role === "owner";

  if (!hasHydrated) {
    return <div className="space-y-3 pt-4">{[1,2].map(i=><Skeleton key={i} className="h-24 rounded-lg"/>)}</div>;
  }

  const { data: authority, isLoading } = useQuery({
    queryKey: ["user-authority"],
    queryFn: userApi.getAuthority,
    retry: false,
  });

  // Refresh JWT if authority is approved but claims are stale (e.g. after admin approval)
  useEffect(() => {
    if (!authority || !user) return;
    const claimsStale =
      (authority.owner_id === user.id && user.authority_role !== "owner") ||
      !user.authority_id;
    if (claimsStale) {
      authApi.refresh().then((updated) => {
        if (updated) setUser(updated);
      });
    }
  }, [authority?.id, user?.id]);

  // Re-render page when JWT is refreshed via WS (e.g. after admin approves authority)
  useEffect(() => {
    function onTokenRefresh() {
      qc.invalidateQueries();
    }
    window.addEventListener("seashell:token_refresh", onTokenRefresh);
    return () => window.removeEventListener("seashell:token_refresh", onTokenRefresh);
  }, [qc]);

  if (isLoading) return <div className="space-y-3 pt-4">{[1,2].map(i=><Skeleton key={i} className="h-24 rounded-lg"/>)}</div>;

  if (!authority) return <NoAuthority />;

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Building2 className="h-5 w-5 text-primary" /> {authority.name}
        </h1>
        <div className="flex items-center gap-2 mt-1">
          <Badge variant={statusColor(authority.status)} className="capitalize">{authority.status}</Badge>
          {user?.authority_role && (
            <Badge variant="outline" className="capitalize border-primary/30 text-primary">{user.authority_role}</Badge>
          )}
        </div>
      </div>

      {authority.description && (
        <p className="text-sm text-muted-foreground">{authority.description}</p>
      )}

      <Tabs defaultValue={isOwner ? "members" : "info"}>
        <TabsList className="w-full">
          <TabsTrigger value="info" className="flex-1">Info</TabsTrigger>
          {isOwner && <TabsTrigger value="members" className="flex-1">Members</TabsTrigger>}
          {isOwner && <TabsTrigger value="invitations" className="flex-1">Invites</TabsTrigger>}
          {isOwner && <TabsTrigger value="stats" className="flex-1">Stats</TabsTrigger>}
          {isOwner && <TabsTrigger value="verification" className="flex-1">Verify</TabsTrigger>}
        </TabsList>

        <TabsContent value="info" className="space-y-3 mt-3">
          <div className="glass-card dark:glass-card glass-card-light rounded-xl p-4 space-y-2 text-sm">
            <Row label="Status" value={<Badge variant={statusColor(authority.status)} className="capitalize">{authority.status}</Badge>} />
            <Row label="Base Price" value={`${authority.base_price} SHELL`} />
            <Row label="Sensitivity" value={authority.sensitivity_k.toString()} />
            {authority.validator_pubkey && (
              <Row label="Validator Key" value={<span className="font-mono text-xs">{truncateHash(authority.validator_pubkey, 10)}</span>} />
            )}
          </div>
        </TabsContent>

        {isOwner && (
          <TabsContent value="members" className="mt-3">
            <MembersTab />
          </TabsContent>
        )}
        {isOwner && (
          <TabsContent value="invitations" className="mt-3">
            <InvitationsTab />
          </TabsContent>
        )}
        {isOwner && (
          <TabsContent value="stats" className="mt-3">
            <StatsTab />
          </TabsContent>
        )}
        {isOwner && (
          <TabsContent value="verification" className="mt-3">
            <VerificationTab />
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between items-center py-1 border-b last:border-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}

function MembersTab() {
  const qc = useQueryClient();
  const { data: members, isLoading } = useQuery({ queryKey: ["authority-members"], queryFn: authorityApi.getMembers });
  const removeMutation = useMutation({
    mutationFn: authorityApi.removeMember,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["authority-members"] }),
  });

  if (isLoading) return <Skeleton className="h-32 rounded-xl" />;
  return (
    <div className="space-y-2">
      {members?.map((m) => (
        <div key={m.id} className="glass-card dark:glass-card glass-card-light rounded-xl flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            <div className="size-9 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold text-sm">
              {m.username[0].toUpperCase()}
            </div>
            <div>
              <p className="font-medium text-sm">{m.username}</p>
              <p className="text-xs text-muted-foreground">{m.email}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="text-xs capitalize border-primary/30 text-primary">{m.authority_role}</Badge>
            {m.authority_role !== "owner" && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-destructive hover:bg-destructive/10"
                onClick={() => removeMutation.mutate(m.id)}
                disabled={removeMutation.isPending}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function InvitationsTab() {
  const qc = useQueryClient();
  const [copiedCode, setCopiedCode] = useState<string | null>(null);
  const { data: invitations, isLoading } = useQuery({ queryKey: ["invitations"], queryFn: authorityApi.getInvitations });
  const createMutation = useMutation({
    mutationFn: authorityApi.createInvitation,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invitations"] }),
  });
  const revokeMutation = useMutation({
    mutationFn: authorityApi.revokeInvitation,
    onSuccess: (_data, code) => {
      qc.setQueryData<typeof invitations>(["invitations"], (old) =>
        old ? old.filter((i) => i.code !== code) : old
      );
    },
  });

  function copyCode(code: string) {
    navigator.clipboard.writeText(code);
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  }

  if (isLoading) return <Skeleton className="h-32 rounded-xl" />;
  return (
    <div className="space-y-3">
      <Button
        onClick={() => createMutation.mutate()}
        disabled={createMutation.isPending}
        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 rounded-xl"
      >
        {createMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Plus className="mr-2 h-4 w-4" />}
        Create Invitation
      </Button>
      {invitations?.map((inv) => (
        <div key={inv.code} className="glass-card dark:glass-card glass-card-light rounded-xl px-4 py-3 flex items-center justify-between">
          <div>
            <p className="font-mono text-sm font-bold text-primary">{inv.code}</p>
            <p className="text-xs text-muted-foreground mt-0.5">
              Expires {new Date(inv.expires_at).toLocaleDateString()}
            </p>
          </div>
          <div className="flex items-center gap-1">
            <Badge variant={inv.used ? "secondary" : "default"} className="text-[10px]">
              {inv.used ? "Used" : "Active"}
            </Badge>
            <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-primary" onClick={() => copyCode(inv.code)}>
              {copiedCode === inv.code ? <Check className="h-3.5 w-3.5 text-primary" /> : <Copy className="h-3.5 w-3.5" />}
            </Button>
            {!inv.used && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-muted-foreground hover:text-destructive"
                onClick={() => revokeMutation.mutate(inv.code)}
                disabled={revokeMutation.isPending}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
        </div>
      ))}
      {(!invitations || invitations.length === 0) && (
        <p className="text-sm text-muted-foreground text-center py-4">No invitations yet.</p>
      )}
    </div>
  );
}

function StatsTab() {
  const { data: stats, isLoading } = useQuery({ queryKey: ["authority-stats"], queryFn: authorityApi.getStats });
  if (isLoading) return <Skeleton className="h-32 rounded-xl" />;
  return (
    <div className="grid grid-cols-3 gap-3">
      {[
        { label: "Members", value: stats?.member_count ?? 0 },
        { label: "Support", value: stats?.ticket_count ?? 0 },
        { label: "Price", value: `${stats?.current_price?.toFixed(4) ?? "0"}` },
      ].map(({ label, value }) => (
        <div key={label} className="glass-card dark:glass-card glass-card-light rounded-xl py-4 text-center">
          <p className="text-2xl font-bold font-mono">{value}</p>
          <p className="text-[10px] text-muted-foreground uppercase tracking-wider mt-1">{label}</p>
        </div>
      ))}
    </div>
  );
}

function VerificationTab() {
  const qc = useQueryClient();
  const [reviewTarget, setReviewTarget] = useState<VerificationSubmission | null>(null);
  const [remarks, setRemarks] = useState("");

  // Config state for form builder
  const [fields, setFields] = useState<FieldDefinition[]>([]);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [optionDrafts, setOptionDrafts] = useState<Record<number, string>>({});

  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ["verify-config"],
    queryFn: verificationApi.getConfig,
    retry: false,
  });

  // Sync config fields into local state once loaded
  if (config && !configLoaded) {
    setFields(config.fields ?? []);
    setConfigLoaded(true);
  }

  const { data: submissions, isLoading: subsLoading } = useQuery({
    queryKey: ["verify-submissions"],
    queryFn: () => verificationApi.listSubmissions(),
  });

  const saveMutation = useMutation({
    mutationFn: (f: FieldDefinition[]) =>
      verificationApi.saveConfig({ method: "manual", fields: f }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["verify-config"] });
    },
  });

  const reviewMutation = useMutation({
    mutationFn: ({ id, action, remarks }: { id: string; action: "approve" | "reject"; remarks?: string }) =>
      verificationApi.reviewSubmission(id, { action, remarks }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["verify-submissions"] });
      setReviewTarget(null);
      setRemarks("");
    },
  });

  function addField() {
    setFields((prev) => [
      ...prev,
      { name: `field_${prev.length + 1}`, label: "", input_type: "text", required: true },
    ]);
  }

  function updateField(index: number, updates: Partial<FieldDefinition>) {
    setFields((prev) => prev.map((f, i) => (i === index ? { ...f, ...updates } : f)));
  }

  function removeField(index: number) {
    setFields((prev) => prev.filter((_, i) => i !== index));
  }

  function autoName(label: string): string {
    return label.toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "");
  }

  const canSave = fields.length > 0 && fields.every((f) => f.label.trim() && f.name.trim());

  const pendingSubs = submissions?.filter((s) => s.status === "pending") ?? [];
  const reviewedSubs = submissions?.filter((s) => s.status !== "pending") ?? [];

  return (
    <div className="space-y-5">
      {/* Config builder */}
      <div className="glass-card dark:glass-card glass-card-light rounded-xl p-5 space-y-4">
        <div>
          <p className="font-semibold flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-primary" /> Verification Setup
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">Define the fields members must fill out for verification.</p>
        </div>
          {configLoading ? (
            <Skeleton className="h-20 rounded-lg" />
          ) : (
            <>
              {fields.map((field, idx) => (
                <div key={idx} className="border rounded-lg p-3 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground font-mono">{field.name || "..."}</span>
                    <Button variant="ghost" size="icon" className="h-6 w-6 text-destructive" onClick={() => removeField(idx)}>
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <Label className="text-xs">Label</Label>
                      <Input
                        placeholder="e.g. PAN Card"
                        value={field.label}
                        onChange={(e) => updateField(idx, { label: e.target.value, name: autoName(e.target.value) || field.name })}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-xs">Input Type</Label>
                      <Select value={field.input_type} onValueChange={(v) => updateField(idx, { input_type: v as FieldDefinition["input_type"] })}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="text">Text</SelectItem>
                          <SelectItem value="number">Number</SelectItem>
                          <SelectItem value="email">Email</SelectItem>
                          <SelectItem value="textarea">Textarea</SelectItem>
                          <SelectItem value="select">Select</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  {field.input_type === "select" ? (
                    <div className="space-y-2">
                      <Label className="text-xs">Options</Label>
                        {(field.options ?? []).length > 0 && (
                          <div className="flex flex-wrap gap-1.5">
                            {field.options!.map((opt, optIdx) => (
                              <span key={optIdx} className="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium">
                                {opt}
                                <button
                                  type="button"
                                  onClick={() => updateField(idx, { options: field.options!.filter((_, i) => i !== optIdx) })}
                                  className="ml-0.5 rounded-full hover:text-destructive"
                                >
                                  <X className="h-3 w-3" />
                                </button>
                              </span>
                            ))}
                          </div>
                        )}
                      <div className="flex gap-2">
                        <Input
                          placeholder="Type an option…"
                          value={optionDrafts[idx] ?? ""}
                          onChange={(e) => setOptionDrafts((p) => ({ ...p, [idx]: e.target.value }))}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              const val = (optionDrafts[idx] ?? "").trim();
                              if (!val || (field.options ?? []).includes(val)) return;
                              updateField(idx, { options: [...(field.options ?? []), val] });
                              setOptionDrafts((p) => ({ ...p, [idx]: "" }));
                            }
                          }}
                        />
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          className="shrink-0"
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={() => {
                            const val = (optionDrafts[idx] ?? "").trim();
                            if (!val || (field.options ?? []).includes(val)) return;
                            updateField(idx, { options: [...(field.options ?? []), val] });
                            setOptionDrafts((p) => ({ ...p, [idx]: "" }));
                          }}
                        >
                          <Plus className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="grid grid-cols-3 gap-3">
                      <div className="space-y-1">
                        <Label className="text-xs">Min Length</Label>
                        <Input
                          type="number"
                          min={0}
                          value={field.min_length ?? ""}
                          onChange={(e) => updateField(idx, { min_length: e.target.value ? parseInt(e.target.value) : 0 })}
                        />
                      </div>
                      <div className="space-y-1">
                        <Label className="text-xs">Max Length</Label>
                        <Input
                          type="number"
                          min={0}
                          value={field.max_length ?? ""}
                          onChange={(e) => updateField(idx, { max_length: e.target.value ? parseInt(e.target.value) : 0 })}
                        />
                      </div>
                      <div className="flex items-end pb-1">
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="checkbox"
                            checked={field.required}
                            onChange={(e) => updateField(idx, { required: e.target.checked })}
                            className="rounded"
                          />
                          Required
                        </label>
                      </div>
                    </div>
                  )}
                  {field.input_type === "select" && (
                    <div className="flex items-center gap-2">
                      <label className="flex items-center gap-2 text-xs cursor-pointer">
                        <input
                          type="checkbox"
                          checked={field.required}
                          onChange={(e) => updateField(idx, { required: e.target.checked })}
                          className="rounded"
                        />
                        Required
                      </label>
                    </div>
                  )}
                  {field.input_type === "number" && (
                    <p className="text-xs text-muted-foreground">Number fields only accept digits.</p>
                  )}
                </div>
              ))}

              <Button variant="outline" size="sm" className="w-full" onClick={addField}>
                <Plus className="mr-2 h-4 w-4" /> Add Field
              </Button>

              {saveMutation.error && (
                <Alert variant="destructive">
                  <AlertDescription>{(saveMutation.error as Error).message}</AlertDescription>
                </Alert>
              )}

              <Button
                className="w-full"
                onClick={() => saveMutation.mutate(fields)}
                disabled={!canSave || saveMutation.isPending}
              >
                {saveMutation.isPending ? "Saving…" : "Save Configuration"}
              </Button>
            </>
          )}
      </div>

      {/* Pending submissions */}
      <div className="glass-card dark:glass-card glass-card-light rounded-xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <p className="font-semibold">Pending Submissions ({pendingSubs.length})</p>
          <Button variant="ghost" size="icon" className="h-7 w-7 text-primary hover:bg-primary/10" onClick={() => qc.invalidateQueries({ queryKey: ["verify-submissions"] })} disabled={subsLoading}>
            <RefreshCw className={`h-3.5 w-3.5 ${subsLoading ? "animate-spin" : ""}`} />
          </Button>
        </div>
        {subsLoading ? (
          <Skeleton className="h-20 rounded-xl" />
        ) : pendingSubs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">No pending submissions.</p>
        ) : (
          <div className="space-y-4">
            {pendingSubs.map((sub) => (
              <div key={sub.id} className="border border-border/50 rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">{sub.full_name || sub.username || sub.user_id}</p>
                    <p className="text-xs text-muted-foreground">Submitted {formatRelative(sub.created_at)}</p>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      className="bg-primary text-primary-foreground hover:bg-primary/90"
                      onClick={() => reviewMutation.mutate({ id: sub.id, action: "approve" })}
                      disabled={reviewMutation.isPending}
                    >
                      Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => { setReviewTarget(sub); setRemarks(""); }}
                    >
                      Reject
                    </Button>
                  </div>
                </div>
                <div className="bg-muted/50 rounded-lg p-3 space-y-1">
                  {Object.entries(sub.field_values).map(([key, val]) => (
                    <div key={key} className="flex justify-between text-xs">
                      <span className="text-muted-foreground">{key}</span>
                      <span className="font-mono">{val}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Recently reviewed */}
      {reviewedSubs.length > 0 && (
        <div className="glass-card dark:glass-card glass-card-light rounded-xl p-5 space-y-3">
          <p className="font-semibold">Reviewed ({reviewedSubs.length})</p>
          <div className="divide-y divide-border/50">
            {reviewedSubs.map((sub) => (
              <div key={sub.id} className="py-3 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium">{sub.full_name || sub.username || sub.user_id}</p>
                  {sub.remarks && <p className="text-xs text-muted-foreground">{sub.remarks}</p>}
                </div>
                <Badge variant={sub.status === "approved" ? "default" : "destructive"} className="capitalize">
                  {sub.status}
                </Badge>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Reject dialog */}
      <Dialog open={!!reviewTarget} onOpenChange={(open) => { if (!open) { setReviewTarget(null); setRemarks(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject Verification</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Label>Remarks (required)</Label>
            <Textarea
              placeholder="Explain what needs to be corrected"
              value={remarks}
              onChange={(e) => setRemarks(e.target.value)}
              rows={3}
            />
            {reviewMutation.error && (
              <Alert variant="destructive">
                <AlertDescription>{(reviewMutation.error as Error).message}</AlertDescription>
              </Alert>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={() => { setReviewTarget(null); setRemarks(""); }}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={!remarks.trim() || reviewMutation.isPending}
              onClick={() => reviewTarget && reviewMutation.mutate({ id: reviewTarget.id, action: "reject", remarks })}
            >
              {reviewMutation.isPending ? "Rejecting…" : "Reject"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function NoAuthority() {
  const qc = useQueryClient();
  const { hasHydrated, user, setUser } = useAuthStore();
  const [tab, setTab] = useState<"create" | "join">("join");
  const [createForm, setCreateForm] = useState({ name: "", description: "" });
  const [joinCode, setJoinCode] = useState("");
  const [error, setError] = useState("");

  const createMutation = useMutation({
    mutationFn: userApi.createAuthority,
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["user-authority"] });
      if (hasHydrated && user) setUser({ ...user, authority_id: data.id, authority_role: "owner" });
    },
    onError: (err: Error) => setError(err.message),
  });

  const joinMutation = useMutation({
    mutationFn: (code: string) => userApi.joinAuthority({ code }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["user-authority"] });
      if (hasHydrated && user) setUser({ ...user, authority_id: data.authority.id, authority_role: "member" });
    },
    onError: (err: Error) => setError(err.message),
  });

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Building2 className="h-5 w-5 text-primary" /> Authority
        </h1>
        <p className="text-sm text-muted-foreground">Join an existing authority or request to create one.</p>
      </div>

      <div className="flex gap-2">
        <Button variant={tab === "join" ? "default" : "outline"} onClick={() => setTab("join")} className="flex-1">Join</Button>
        <Button variant={tab === "create" ? "default" : "outline"} onClick={() => setTab("create")} className="flex-1">Create</Button>
      </div>

      {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

      {tab === "join" ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Join with Code</CardTitle>
            <CardDescription>Enter the invitation code from an authority owner.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Input
              placeholder="e.g. a1b2c3d4e5f6"
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value)}
              className="font-mono"
            />
            <Button
              className="w-full"
              onClick={() => { setError(""); joinMutation.mutate(joinCode); }}
              disabled={joinMutation.isPending || !joinCode}
            >
              {joinMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Join Authority
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Request New Authority</CardTitle>
            <CardDescription>Admins will review and approve your request.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Authority Name</Label>
              <Input
                placeholder="My Authority"
                value={createForm.name}
                onChange={(e) => setCreateForm((f) => ({ ...f, name: e.target.value }))}
                minLength={3}
              />
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Textarea
                placeholder="What does your authority do?"
                value={createForm.description}
                onChange={(e) => setCreateForm((f) => ({ ...f, description: e.target.value }))}
                rows={3}
              />
            </div>
            <Button
              className="w-full"
              onClick={() => { setError(""); createMutation.mutate(createForm); }}
              disabled={createMutation.isPending || createForm.name.length < 3}
            >
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Submit Request
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
