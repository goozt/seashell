"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi, nodeApi } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ShieldCheck, Users, Building2, TicketCheck, Network } from "lucide-react";
import Link from "next/link";

export default function AdminDashboard() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: adminApi.getStats,
  });

  const { data: nodesData, isLoading: nodesLoading } = useQuery({
    queryKey: ["admin", "nodes"],
    queryFn: () => nodeApi.getNodes(),
    refetchInterval: 30_000,
  });

  const nodes = nodesData?.nodes ?? [];
  const activeCount = nodes.filter((n) => n.status === "active").length;
  const totalCount = nodes.length;

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

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
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

        {/* Network status card */}
        <Link href="/admin/network">
          <Card className="hover:bg-muted/50 transition-colors cursor-pointer h-full">
            <CardContent className="py-4 text-center">
              <Network className="h-5 w-5 mx-auto mb-1 text-primary" />
              {nodesLoading ? (
                <Skeleton className="h-7 w-16 mx-auto" />
              ) : (
                <p className="text-2xl font-bold">
                  <span className="text-green-600">{activeCount}</span>
                  <span className="text-muted-foreground text-base">/{totalCount}</span>
                </p>
              )}
              <p className="text-xs text-muted-foreground mt-0.5">Nodes Online</p>
              {!nodesLoading && nodes.length > 0 && (
                <div className="flex justify-center gap-1 mt-2 flex-wrap">
                  {nodes.slice(0, 8).map((n) => (
                    <span
                      key={n.id}
                      title={`${n.authority_name}: ${n.status}`}
                      className={`inline-block w-2.5 h-2.5 rounded-full ${
                        n.status === "active"
                          ? "bg-green-500"
                          : n.status === "offline"
                            ? "bg-red-500"
                            : n.status === "pending"
                              ? "bg-yellow-500"
                              : "bg-gray-400"
                      }`}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </Link>
      </div>
    </div>
  );
}
