"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { superAdminApi, adminApi } from "@/lib/api";
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
import { Star, ShieldCheck, Building2, Users, Plus, Trash2, Loader2, PauseCircle, PlayCircle } from "lucide-react";

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

  const statItems = [
    { label: "Users", value: stats?.total_users, icon: Users },
    { label: "Admins", value: stats?.total_admins, icon: ShieldCheck },
    { label: "Authorities", value: stats?.total_authorities, icon: Building2 },
    { label: "Active", value: stats?.active_authorities, icon: Star },
    { label: "Pending", value: stats?.pending_authorities, icon: Star },
    { label: "Tickets", value: stats?.total_tickets, icon: Star },
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
      </Tabs>
    </div>
  );
}
