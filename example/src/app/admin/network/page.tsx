"use client";

import { useQuery } from "@tanstack/react-query";
import { nodeApi } from "@/lib/api";
import type { NetworkNode, NetworkStats } from "@/types/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";

function nodeStatusColor(status: string): string {
  switch (status) {
    case "active":
      return "bg-green-500";
    case "offline":
      return "bg-red-500";
    case "pending":
      return "bg-yellow-500";
    case "suspended":
      return "bg-gray-400";
    default:
      return "bg-gray-300";
  }
}

function NodeStatusBadge({ status }: { status: string }) {
  const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
    active: "default",
    offline: "destructive",
    pending: "secondary",
    suspended: "outline",
    rejected: "outline",
  };
  return (
    <Badge variant={variants[status] ?? "outline"}>
      <span className={`inline-block w-2 h-2 rounded-full mr-1 ${nodeStatusColor(status)}`} />
      {status}
    </Badge>
  );
}

function relativeTime(ts?: string): string {
  if (!ts) return "never";
  const diff = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function computeStats(nodes: NetworkNode[]): NetworkStats {
  return {
    total: nodes.length,
    active: nodes.filter((n) => n.status === "active").length,
    offline: nodes.filter((n) => n.status === "offline").length,
    pending: nodes.filter((n) => n.status === "pending").length,
  };
}

export default function NetworkPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin", "nodes"],
    queryFn: () => nodeApi.getNodes(),
    refetchInterval: 30_000,
  });

  const nodes: NetworkNode[] = data?.nodes ?? [];
  const stats = computeStats(nodes);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Network Overview</h1>
        <p className="text-muted-foreground text-sm mt-1">
          Live status of all nodes in the SeaShell blockchain network. Refreshes every 30 seconds.
        </p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: "Total Nodes", value: stats.total, color: "text-foreground" },
          { label: "Active", value: stats.active, color: "text-green-600" },
          { label: "Offline", value: stats.offline, color: "text-red-600" },
          { label: "Pending", value: stats.pending, color: "text-yellow-600" },
        ].map(({ label, value, color }) => (
          <Card key={label}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className={`text-3xl font-bold ${color}`}>
                {isLoading ? <Skeleton className="h-8 w-10" /> : value}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Node table */}
      <Card>
        <CardHeader>
          <CardTitle>Nodes</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(4)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : nodes.length === 0 ? (
            <p className="text-muted-foreground text-sm">No nodes registered yet.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="text-left py-2 pr-4 font-medium">Node</th>
                    <th className="text-left py-2 pr-4 font-medium">Status</th>
                    <th className="text-left py-2 pr-4 font-medium">Height</th>
                    <th className="text-left py-2 pr-4 font-medium">URL</th>
                    <th className="text-left py-2 font-medium">Last Seen</th>
                  </tr>
                </thead>
                <tbody>
                  {nodes.map((n) => (
                    <tr key={n.id} className="border-b hover:bg-muted/50 transition-colors">
                      <td className="py-3 pr-4">
                        <Link
                          href={`/admin/network/${n.id}`}
                          className="font-medium hover:underline"
                        >
                          {n.is_primary && (
                            <span className="mr-1" title="Primary node">★</span>
                          )}
                          {n.authority_name}
                        </Link>
                      </td>
                      <td className="py-3 pr-4">
                        <NodeStatusBadge status={n.status} />
                      </td>
                      <td className="py-3 pr-4 font-mono">
                        {n.block_height.toLocaleString()}
                      </td>
                      <td className="py-3 pr-4">
                        <a
                          href={n.node_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:underline truncate max-w-[180px] inline-block"
                        >
                          {n.node_url}
                        </a>
                      </td>
                      <td className="py-3 text-muted-foreground">
                        {relativeTime(n.last_seen_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
