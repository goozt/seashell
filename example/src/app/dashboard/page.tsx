"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/store/auth";
import { userApi, publicApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { statusColor } from "@/lib/utils";
import Link from "next/link";
import {
  Wallet,
  Building2,
  Layers,
  TrendingUp,
  ArrowRight,
  Plus,
} from "lucide-react";

export default function DashboardPage() {
  const { user } = useAuthStore();

  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ["wallet"],
    queryFn: userApi.getWallet,
    retry: false,
  });

  const { data: authority, isLoading: authorityLoading } = useQuery({
    queryKey: ["user-authority"],
    queryFn: userApi.getAuthority,
    retry: false,
  });

  const { data: blocks } = useQuery({
    queryKey: ["blocks"],
    queryFn: publicApi.getBlocks,
    refetchInterval: 15_000,
  });

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold">Welcome back, {user?.username}</h1>
        <p className="text-sm text-muted-foreground">Here's your ecosystem overview.</p>
      </div>

      {/* Wallet card */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1">
              <Wallet className="h-4 w-4" /> Wallet
            </CardTitle>
            <Button asChild variant="ghost" size="sm" className="h-7 text-xs">
              <Link href="/dashboard/wallet">
                Manage <ArrowRight className="ml-1 h-3 w-3" />
              </Link>
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {walletLoading ? (
            <Skeleton className="h-8 w-32" />
          ) : wallet ? (
            <div>
              <p className="text-2xl font-bold">{wallet.balance} <span className="text-base font-normal text-muted-foreground">SHELL</span></p>
              <p className="text-xs text-muted-foreground font-mono mt-1 truncate">{wallet.address}</p>
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <p className="text-sm text-muted-foreground">No wallet yet.</p>
              <Button asChild size="sm" variant="outline">
                <Link href="/dashboard/wallet">
                  <Plus className="mr-1 h-3 w-3" /> Create
                </Link>
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Authority card */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1">
              <Building2 className="h-4 w-4" /> Authority
            </CardTitle>
            <Button asChild variant="ghost" size="sm" className="h-7 text-xs">
              <Link href="/dashboard/authority">
                View <ArrowRight className="ml-1 h-3 w-3" />
              </Link>
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {authorityLoading ? (
            <Skeleton className="h-6 w-40" />
          ) : authority ? (
            <div className="flex items-center gap-2">
              <p className="font-semibold">{authority.name}</p>
              <Badge variant={statusColor(authority.status)} className="capitalize text-xs">
                {authority.status}
              </Badge>
              {user?.authority_role && (
                <Badge variant="outline" className="text-xs capitalize">{user.authority_role}</Badge>
              )}
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <p className="text-sm text-muted-foreground">Not affiliated.</p>
              <Button asChild size="sm" variant="outline">
                <Link href="/dashboard/authority">Join or Create</Link>
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Recent blocks */}
      <div>
        <div className="flex items-center gap-2 mb-2">
          <Layers className="h-4 w-4 text-primary" />
          <h2 className="text-sm font-semibold">Recent Blocks</h2>
        </div>
        <div className="space-y-1.5">
          {blocks?.slice(0, 5).map((b) => (
            <Card key={b.hash}>
              <CardContent className="flex items-center justify-between py-2 px-4">
                <div>
                  <span className="text-sm font-medium">Block #{b.height}</span>
                  <span className="text-xs text-muted-foreground ml-2">{b.tx_count} txns</span>
                </div>
                <Badge variant={b.valid_poa ? "default" : "destructive"} className="text-xs">
                  {b.valid_poa ? "Valid" : "Invalid"}
                </Badge>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}
