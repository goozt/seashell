"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ShieldCheck, Users, Building2, TicketCheck } from "lucide-react";

export default function AdminDashboard() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: adminApi.getStats,
  });

  const statItems = [
    { label: "Total Users", value: stats?.total_users, icon: Users },
    { label: "Authorities", value: stats?.total_authorities, icon: Building2 },
    { label: "Tickets", value: stats?.total_tickets, icon: TicketCheck },
  ];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" /> Admin Dashboard
        </h1>
        <p className="text-sm text-muted-foreground">Platform overview and management.</p>
      </div>

      <div className="grid grid-cols-3 gap-3">
        {statItems.map(({ label, value, icon: Icon }) => (
          <Card key={label}>
            <CardContent className="py-4 text-center">
              <Icon className="h-5 w-5 mx-auto mb-1 text-primary" />
              {isLoading ? (
                <Skeleton className="h-7 w-12 mx-auto" />
              ) : (
                <p className="text-2xl font-bold">{value ?? 0}</p>
              )}
              <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
