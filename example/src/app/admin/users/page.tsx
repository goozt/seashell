"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Users } from "lucide-react";
import { useState } from "react";

export default function AdminUsersPage() {
  const [search, setSearch] = useState("");

  const { data: users, isLoading } = useQuery({
    queryKey: ["admin-users"],
    queryFn: adminApi.getUsers,
  });

  const filtered = users?.filter(
    (u) =>
      u.username.toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Users className="h-5 w-5 text-primary" /> Users
        </h1>
        <p className="text-sm text-muted-foreground">{users?.length ?? 0} total users.</p>
      </div>

      <Input
        placeholder="Search by name or email…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />

      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-16 rounded-lg" />)}
        </div>
      ) : (
        <div className="space-y-2">
          {filtered?.map((user) => (
            <Card key={user.id}>
              <CardContent className="flex items-center justify-between py-3 px-4">
                <div className="min-w-0">
                  <p className="font-medium text-sm">{user.username}</p>
                  <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                  {user.authority_id && (
                    <p className="text-xs text-muted-foreground capitalize">
                      {user.authority_role} of authority
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-1.5 ml-2 shrink-0">
                  {user.role !== "user" && (
                    <Badge variant="secondary" className="text-xs capitalize">{user.role}</Badge>
                  )}
                  <Badge variant={user.active ? "default" : "destructive"} className="text-xs">
                    {user.active ? "Active" : "Inactive"}
                  </Badge>
                </div>
              </CardContent>
            </Card>
          ))}
          {filtered?.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-6">No users match your search.</p>
          )}
        </div>
      )}
    </div>
  );
}
