"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { superAdminApi, adminApi, nodeApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { statusColor } from "@/lib/utils";
import {
  Star, ShieldCheck, Building2, Users, Plus, Trash2, Loader2,
  PauseCircle, PlayCircle, Network, CheckCircle2, XCircle,
} from "lucide-react";
import type { NodeJoinRequest } from "@/types/api";

function relativeTime(ts?: string): string {
  if (!ts) return "—";
  const diff = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return new Date(ts).toLocaleDateString();
}

function NodeRequestRow({
  req,
  onApprove,
  onReject,
  loading,
}: {
  req: NodeJoinRequest;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  loading: boolean;
}) {
  const [rejectOpen, setRejectOpen] = useState(false);
  const [reason, setReason] = useState("");

  return (
    <Card>
      <CardContent className="py-3 px-4 space-y-2">
        <div className="flex items-start justify-between">
          <div>
            <p className="font-medium text-sm">{req.authority_name}</p>
            <p className="text-xs text-muted-foreground truncate max-w-[200px]">{req.node_url}</p>
            <p className="text-xs text-muted-foreground">{req.admin_email}</p>
          </div>
          <div className="text-right">
            <Badge variant={req.status === "pending" ? "secondary" : req.status === "approved" ? "default" : "destructive"}>
              {req.status}
            </Badge>
            <p className="text-xs text-muted-foreground mt-1">{relativeTime(req.created_at)}</p>
          </div>
        </div>

        {req.status === "pending" && (
          <div className="flex gap-2">
            <Button
              size="sm"
              className="flex-1"
              onClick={() => onApprove(req.id)}
              disabled={loading}
            >
              <CheckCircle2 className="mr-1 h-4 w-4" /> Approve
            </Button>
            <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
              <DialogTrigger asChild>
                <Button size="sm" variant="outline" className="flex-1">
                  <XCircle className="mr-1 h-4 w-4 text-destructive" /> Reject
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-sm mx-auto">
                <DialogHeader>
                  <DialogTitle>Reject Node Request</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 mt-2">
                  <div className="space-y-2">
                    <Label>Reason (optional)</Label>
                    <Input
                      placeholder="e.g. Unable to verify authority"
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                    />
                  </div>
                  <Button
                    variant="destructive"
                    className="w-full"
                    onClick={() => { onReject(req.id, reason); setRejectOpen(false); }}
                    disabled={loading}
                  >
                    Confirm Rejection
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function SuperAdminPage() {
  const qc = useQueryClient();

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ["superadmin-stats"],
    queryFn: superAdminApi.getStats,
  });

  const { data: admins, isLoading: adminsLoading } = useQuery({
    queryKey: ["admins"],
    queryFn: superAdminApi.getAdmins,
  });

  const { data: authorities, isLoading: authoritiesLoading } = useQuery({
    queryKey: ["all-authorities"],
    queryFn: adminApi.getAuthorities,
  });

  const { data: nodeReqData, isLoading: nodeReqLoading } = useQuery({
    queryKey: ["superadmin", "node-requests"],
    queryFn: nodeApi.getNodeRequests,
    refetchInterval: 30_000,
  });

  const nodeRequests: NodeJoinRequest[] = nodeReqData?.requests ?? [];
  const pendingNodeCount = nodeRequests.filter((r) => r.status === "pending").length;

  const [newAdmin, setNewAdmin] = useState({ username: "", email: "", password: "" });
  const [adminDialogOpen, setAdminDialogOpen] = useState(false);
  const [adminError, setAdminError] = useState("");

  const createAdminMutation = useMutation({
    mutationFn: superAdminApi.createAdmin,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admins"] });
      qc.invalidateQueries({ queryKey: ["superadmin-stats"] });
      setAdminDialogOpen(false);
      setNewAdmin({ username: "", email: "", password: "" });
    },
    onError: (err: Error) => setAdminError(err.message),
  });

  const demoteMutation = useMutation({
    mutationFn: superAdminApi.demoteAdmin,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admins"] });
      qc.invalidateQueries({ queryKey: ["superadmin-stats"] });
    },
  });

  const suspendMutation = useMutation({
    mutationFn: superAdminApi.suspendAuthority,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["all-authorities"] }),
  });

  const reinstateMutation = useMutation({
    mutationFn: superAdminApi.reinstateAuthority,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["all-authorities"] }),
  });

  const approveNodeMutation = useMutation({
    mutationFn: (id: string) => nodeApi.approveNodeRequest(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["superadmin", "node-requests"] }),
  });

  const rejectNodeMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      nodeApi.rejectNodeRequest(id, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["superadmin", "node-requests"] }),
  });

  const statItems = [
    { label: "Users", value: stats?.total_users, icon: Users },
    { label: "Admins", value: stats?.total_admins, icon: ShieldCheck },
    { label: "Authorities", value: stats?.total_authorities, icon: Building2 },
    { label: "Active", value: stats?.active_authorities, icon: Star },
    { label: "Pending", value: stats?.pending_authorities, icon: Star },
    { label: "Support", value: stats?.total_tickets, icon: Star },
  ];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Star className="h-5 w-5 text-primary" /> Super Admin
        </h1>
        <p className="text-sm text-muted-foreground">Full platform control.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-2">
        {statItems.map(({ label, value, icon: Icon }) => (
          <Card key={label}>
            <CardContent className="py-3 text-center">
              {statsLoading ? (
                <Skeleton className="h-6 w-8 mx-auto" />
              ) : (
                <p className="text-xl font-bold">{value ?? 0}</p>
              )}
              <p className="text-xs text-muted-foreground">{label}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs defaultValue="admins">
        <TabsList className="w-full">
          <TabsTrigger value="admins" className="flex-1">Admins</TabsTrigger>
          <TabsTrigger value="authorities" className="flex-1">Authorities</TabsTrigger>
          <TabsTrigger value="node-requests" className="flex-1 relative">
            <Network className="mr-1 h-3.5 w-3.5" />
            Nodes
            {pendingNodeCount > 0 && (
              <span className="ml-1 bg-primary text-primary-foreground text-[10px] rounded-full px-1.5 py-0.5">
                {pendingNodeCount}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Admins Tab */}
        <TabsContent value="admins" className="space-y-3 mt-3">
          <div className="flex justify-end">
            <Dialog open={adminDialogOpen} onOpenChange={setAdminDialogOpen}>
              <DialogTrigger asChild>
                <Button size="sm"><Plus className="mr-1 h-4 w-4" /> New Admin</Button>
              </DialogTrigger>
              <DialogContent className="max-w-sm mx-auto">
                <DialogHeader>
                  <DialogTitle>Create Admin Account</DialogTitle>
                </DialogHeader>
                <div className="space-y-4 mt-2">
                  {adminError && <Alert variant="destructive"><AlertDescription>{adminError}</AlertDescription></Alert>}
                  {(["username", "email", "password"] as const).map((field) => (
                    <div key={field} className="space-y-2">
                      <Label className="capitalize">{field}</Label>
                      <Input
                        type={field === "password" ? "password" : field === "email" ? "email" : "text"}
                        placeholder={field === "username" ? "adminuser" : field === "email" ? "admin@example.com" : "min 8 chars"}
                        value={newAdmin[field]}
                        onChange={(e) => setNewAdmin((f) => ({ ...f, [field]: e.target.value }))}
                      />
                    </div>
                  ))}
                  <Button
                    className="w-full"
                    onClick={() => { setAdminError(""); createAdminMutation.mutate(newAdmin); }}
                    disabled={createAdminMutation.isPending}
                  >
                    {createAdminMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Create Admin
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>

          {adminsLoading ? (
            <div className="space-y-2">
              {[1, 2].map((i) => <Skeleton key={i} className="h-14 rounded-lg" />)}
            </div>
          ) : admins?.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">No admins yet.</p>
          ) : (
            <div className="space-y-2">
              {admins?.map((admin) => (
                <Card key={admin.id}>
                  <CardContent className="flex items-center justify-between py-3 px-4">
                    <div>
                      <p className="font-medium text-sm">{admin.username}</p>
                      <p className="text-xs text-muted-foreground">{admin.email}</p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={() => demoteMutation.mutate(admin.id)}
                      disabled={demoteMutation.isPending}
                      title="Demote to user"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Authorities Tab */}
        <TabsContent value="authorities" className="space-y-2 mt-3">
          {authoritiesLoading ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-14 rounded-lg" />)}
            </div>
          ) : authorities?.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">No authorities.</p>
          ) : (
            <div className="space-y-2">
              {authorities?.map((auth) => (
                <Card key={auth.id}>
                  <CardContent className="flex items-center justify-between py-3 px-4">
                    <div>
                      <p className="font-medium text-sm">{auth.name}</p>
                      <Badge variant={statusColor(auth.status)} className="text-xs capitalize mt-1">{auth.status}</Badge>
                    </div>
                    <div className="flex items-center gap-1">
                      {auth.status === "active" && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive"
                          onClick={() => suspendMutation.mutate(auth.id)}
                          disabled={suspendMutation.isPending}
                          title="Suspend"
                        >
                          <PauseCircle className="h-4 w-4" />
                        </Button>
                      )}
                      {auth.status === "suspended" && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-green-600"
                          onClick={() => reinstateMutation.mutate(auth.id)}
                          disabled={reinstateMutation.isPending}
                          title="Reinstate"
                        >
                          <PlayCircle className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Node Requests Tab (primary node only) */}
        <TabsContent value="node-requests" className="space-y-3 mt-3">
          <p className="text-xs text-muted-foreground">
            New authorities submit join requests here. Approving adds their validator key to the
            blockchain network and distributes it to all active nodes.
          </p>

          {nodeReqLoading ? (
            <div className="space-y-2">
              {[1, 2].map((i) => <Skeleton key={i} className="h-28 rounded-lg" />)}
            </div>
          ) : nodeRequests.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">No node requests yet.</p>
          ) : (
            <div className="space-y-3">
              {nodeRequests.map((req) => (
                <NodeRequestRow
                  key={req.id}
                  req={req}
                  loading={approveNodeMutation.isPending || rejectNodeMutation.isPending}
                  onApprove={(id) => approveNodeMutation.mutate(id)}
                  onReject={(id, reason) => rejectNodeMutation.mutate({ id, reason })}
                />
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
