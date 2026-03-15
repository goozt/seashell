"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userApi, authorityApi } from "@/lib/api";
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
import { statusColor, truncateHash } from "@/lib/utils";
import { Building2, Users, Clipboard, Plus, Trash2, Loader2, Copy, Check } from "lucide-react";

export default function AuthorityPage() {
  const qc = useQueryClient();
  const { user } = useAuthStore();
  const isOwner = user?.authority_role === "owner";

  const { data: authority, isLoading } = useQuery({
    queryKey: ["user-authority"],
    queryFn: userApi.getAuthority,
    retry: false,
  });

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
          {user?.authority_role && <Badge variant="outline" className="capitalize">{user.authority_role}</Badge>}
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
        </TabsList>

        <TabsContent value="info" className="space-y-3 mt-3">
          <Card>
            <CardContent className="py-4 space-y-2 text-sm">
              <Row label="Status" value={<Badge variant={statusColor(authority.status)} className="capitalize">{authority.status}</Badge>} />
              <Row label="Base Price" value={`${authority.base_price} SHELL`} />
              <Row label="Sensitivity" value={authority.sensitivity_k.toString()} />
              {authority.validator_pubkey && (
                <Row label="Validator Key" value={<span className="font-mono text-xs">{truncateHash(authority.validator_pubkey, 10)}</span>} />
              )}
            </CardContent>
          </Card>
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

  if (isLoading) return <Skeleton className="h-32 rounded-lg" />;
  return (
    <div className="space-y-2">
      {members?.map((m) => (
        <Card key={m.id}>
          <CardContent className="flex items-center justify-between py-3 px-4">
            <div>
              <p className="font-medium text-sm">{m.username}</p>
              <p className="text-xs text-muted-foreground">{m.email}</p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="text-xs capitalize">{m.authority_role}</Badge>
              {m.authority_role !== "owner" && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-destructive"
                  onClick={() => removeMutation.mutate(m.id)}
                  disabled={removeMutation.isPending}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
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
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invitations"] }),
  });

  function copyCode(code: string) {
    navigator.clipboard.writeText(code);
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  }

  if (isLoading) return <Skeleton className="h-32 rounded-lg" />;
  return (
    <div className="space-y-3">
      <Button onClick={() => createMutation.mutate()} disabled={createMutation.isPending} size="sm" className="w-full">
        {createMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Plus className="mr-2 h-4 w-4" />}
        Create Invitation
      </Button>
      {invitations?.map((inv) => (
        <Card key={inv.code}>
          <CardContent className="py-3 px-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="font-mono text-sm font-bold">{inv.code}</span>
                <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => copyCode(inv.code)}>
                  {copiedCode === inv.code ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                </Button>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={inv.used ? "secondary" : "default"} className="text-xs">
                  {inv.used ? "Used" : "Active"}
                </Badge>
                {!inv.used && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive"
                    onClick={() => revokeMutation.mutate(inv.code)}
                    disabled={revokeMutation.isPending}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                )}
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">Expires: {new Date(inv.expires_at).toLocaleDateString()}</p>
          </CardContent>
        </Card>
      ))}
      {(!invitations || invitations.length === 0) && (
        <p className="text-sm text-muted-foreground text-center py-4">No invitations yet.</p>
      )}
    </div>
  );
}

function StatsTab() {
  const { data: stats, isLoading } = useQuery({ queryKey: ["authority-stats"], queryFn: authorityApi.getStats });
  if (isLoading) return <Skeleton className="h-32 rounded-lg" />;
  return (
    <div className="grid grid-cols-3 gap-3">
      {[
        { label: "Members", value: stats?.member_count ?? 0 },
        { label: "Tickets", value: stats?.ticket_count ?? 0 },
        { label: "Price", value: `${stats?.current_price?.toFixed(4) ?? "0"} SHELL` },
      ].map(({ label, value }) => (
        <Card key={label}>
          <CardContent className="py-4 text-center">
            <p className="text-2xl font-bold">{value}</p>
            <p className="text-xs text-muted-foreground mt-1">{label}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function NoAuthority() {
  const qc = useQueryClient();
  const { user, setUser } = useAuthStore();
  const [tab, setTab] = useState<"create" | "join">("join");
  const [createForm, setCreateForm] = useState({ name: "", description: "" });
  const [joinCode, setJoinCode] = useState("");
  const [error, setError] = useState("");

  const createMutation = useMutation({
    mutationFn: userApi.createAuthority,
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["user-authority"] });
      if (user) setUser({ ...user, authority_id: data.id, authority_role: "owner" });
    },
    onError: (err: Error) => setError(err.message),
  });

  const joinMutation = useMutation({
    mutationFn: (code: string) => userApi.joinAuthority({ code }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["user-authority"] });
      if (user) setUser({ ...user, authority_id: data.authority.id, authority_role: "member" });
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
