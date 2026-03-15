"use client";

import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { nodeApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import Link from "next/link";

function relativeTime(ts?: string): string {
  if (!ts) return "never";
  const diff = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return new Date(ts).toLocaleDateString();
}

function truncate(s?: string, n = 32): string {
  if (!s) return "—";
  return s.length > n ? s.slice(0, n) + "..." : s;
}

export default function NodeDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const { user } = useAuthStore();
  const isSuperAdmin = user?.role === "superadmin";
  const qc = useQueryClient();

  const { data: node, isLoading, error } = useQuery({
    queryKey: ["admin", "nodes", id],
    queryFn: () => nodeApi.getNode(id),
    refetchInterval: 30_000,
  });

  const suspendMutation = useMutation({
    mutationFn: () => nodeApi.suspendNode(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "nodes"] }),
  });

  const reinstateMutation = useMutation({
    mutationFn: () => nodeApi.reinstateNode(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "nodes"] }),
  });

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (error || !node) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertDescription>Node not found or an error occurred.</AlertDescription>
        </Alert>
      </div>
    );
  }

  const statusVariant =
    node.status === "active"
      ? "default"
      : node.status === "offline"
        ? "destructive"
        : "secondary";

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <Link href="/admin/network" className="text-muted-foreground hover:underline text-sm">
          ← Back to Network
        </Link>
      </div>

      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {node.is_primary && <span className="mr-1" title="Primary node">★</span>}
            {node.authority_name}
          </h1>
          <p className="text-muted-foreground text-sm mt-1">Node ID: {truncate(node.id, 40)}</p>
        </div>
        <Badge variant={statusVariant}>{node.status}</Badge>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Node URL</CardTitle>
          </CardHeader>
          <CardContent>
            <a
              href={node.node_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 hover:underline break-all text-sm"
            >
              {node.node_url}
            </a>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Block Height</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{node.block_height.toLocaleString()}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Last Seen</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">{relativeTime(node.last_seen_at)}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Joined Network</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">{new Date(node.registered_at).toLocaleDateString()}</p>
            {node.approved_at && (
              <p className="text-xs text-muted-foreground">
                Approved {new Date(node.approved_at).toLocaleDateString()}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {node.validator_pubkey && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Validator Public Key</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="font-mono text-xs break-all text-muted-foreground">
              {node.validator_pubkey}
            </p>
          </CardContent>
        </Card>
      )}

      {isSuperAdmin && !node.is_primary && (
        <div className="flex gap-3">
          {node.status === "active" ? (
            <Button
              variant="destructive"
              onClick={() => suspendMutation.mutate()}
              disabled={suspendMutation.isPending}
            >
              {suspendMutation.isPending ? "Suspending..." : "Suspend Node"}
            </Button>
          ) : node.status === "suspended" ? (
            <Button
              variant="default"
              onClick={() => reinstateMutation.mutate()}
              disabled={reinstateMutation.isPending}
            >
              {reinstateMutation.isPending ? "Reinstating..." : "Reinstate Node"}
            </Button>
          ) : null}
        </div>
      )}
    </div>
  );
}
