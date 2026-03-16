"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@/store/auth";
import { userApi, publicApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { statusColor, truncateHash } from "@/lib/utils";
import Link from "next/link";
import {
  Wallet,
  Building2,
  Layers,
  ArrowRight,
  Plus,
  ArrowUpRight,
  ArrowDownLeft,
  Copy,
} from "lucide-react";
import { useState } from "react";

export default function DashboardPage() {
  const { hasHydrated, user } = useAuthStore();
  const [copied, setCopied] = useState(false);

  if (!hasHydrated) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-40 w-full rounded-2xl" />
        <div className="grid grid-cols-2 gap-3">
          <Skeleton className="h-12 rounded-xl" />
          <Skeleton className="h-12 rounded-xl" />
        </div>
        <Skeleton className="h-32 w-full rounded-2xl" />
      </div>
    );
  }

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

  function copyAddress() {
    if (!wallet?.address) return;
    navigator.clipboard.writeText(wallet.address);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold">Welcome, {user?.first_name || user?.username}</h1>
        <p className="text-sm text-muted-foreground">Your ecosystem overview.</p>
      </div>

      {/* Wallet balance card */}
      <div className="glass-card dark:glass-card glass-card-light rounded-2xl p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Wallet className="h-4 w-4 text-primary" />
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Main Wallet Balance</span>
          </div>
          <Button asChild variant="ghost" size="sm" className="h-7 text-xs text-primary hover:bg-primary/10 -mr-2">
            <Link href="/dashboard/wallet">
              Manage <ArrowRight className="ml-1 h-3 w-3" />
            </Link>
          </Button>
        </div>

        {walletLoading ? (
          <Skeleton className="h-10 w-48" />
        ) : wallet ? (
          <>
            <p className="text-4xl font-bold font-mono">
              {wallet.balance}{" "}
              <span className="text-lg font-normal text-primary">SHELL</span>
            </p>
            <button
              onClick={copyAddress}
              className="flex items-center gap-1.5 mt-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <span className="font-mono">{truncateHash(wallet.address, 8)}</span>
              <Copy className="h-3 w-3" />
              {copied && <span className="text-primary text-[10px]">Copied!</span>}
            </button>
          </>
        ) : (
          <div className="flex items-center gap-3">
            <p className="text-sm text-muted-foreground">No wallet yet.</p>
            <Button asChild size="sm" className="bg-primary text-primary-foreground hover:bg-primary/90">
              <Link href="/dashboard/wallet">
                <Plus className="mr-1 h-3 w-3" /> Create
              </Link>
            </Button>
          </div>
        )}

        {/* Quick actions */}
        {wallet && (
          <div className="grid grid-cols-2 gap-3 mt-4">
            <Button
              asChild
              className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-xl font-bold"
            >
              <Link href="/dashboard/transactions">
                <ArrowUpRight className="mr-2 h-4 w-4" /> Send
              </Link>
            </Button>
            <Button
              variant="secondary"
              className="rounded-xl font-bold"
              onClick={copyAddress}
            >
              <ArrowDownLeft className="mr-2 h-4 w-4" /> Receive
            </Button>
          </div>
        )}
      </div>

      {/* Authority card */}
      <div className="glass-card dark:glass-card glass-card-light rounded-2xl p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Building2 className="h-4 w-4 text-primary" />
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Authority</span>
          </div>
          <Button asChild variant="ghost" size="sm" className="h-7 text-xs text-primary hover:bg-primary/10 -mr-2">
            <Link href="/dashboard/authority">
              View <ArrowRight className="ml-1 h-3 w-3" />
            </Link>
          </Button>
        </div>

        {authorityLoading ? (
          <Skeleton className="h-6 w-40" />
        ) : authority ? (
          <div className="flex flex-wrap items-center gap-2">
            <p className="font-semibold">{authority.name}</p>
            <Badge variant={statusColor(authority.status)} className="capitalize text-xs">
              {authority.status}
            </Badge>
            {user?.authority_role && (
              <Badge variant="outline" className="text-xs capitalize border-primary/30 text-primary">
                {user.authority_role}
              </Badge>
            )}
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <p className="text-sm text-muted-foreground">Not affiliated.</p>
            <Button asChild size="sm" className="bg-primary text-primary-foreground hover:bg-primary/90">
              <Link href="/dashboard/authority">Join or Create</Link>
            </Button>
          </div>
        )}
      </div>

      {/* Recent blocks */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <Layers className="h-4 w-4 text-primary" />
          <h2 className="text-sm font-bold uppercase tracking-wider">Recent Blocks</h2>
        </div>
        <div className="space-y-2">
          {blocks?.slice(0, 5).map((b, idx) => (
            <div
              key={b.hash}
              className={`glass-card dark:glass-card glass-card-light p-3 rounded-xl flex items-center justify-between border-l-4 ${
                idx === 0 ? "border-l-primary" : "border-l-primary/20"
              }`}
            >
              <div>
                <span className="text-xs text-muted-foreground">Block #{b.height}</span>
                <p className="font-mono text-xs text-foreground/70 truncate w-40">{truncateHash(b.hash, 8)}</p>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">{b.tx_count} TXs</span>
                <Badge variant={b.valid_poa ? "default" : "destructive"} className="text-[10px]">
                  {b.valid_poa ? "Valid" : "Invalid"}
                </Badge>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
