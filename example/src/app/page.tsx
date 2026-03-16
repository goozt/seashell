"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { publicApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { truncateHash, formatRelative, statusColor } from "@/lib/utils";
import { TrendingUp, Layers, LogIn, LayoutDashboard } from "lucide-react";
import { LogoWordmark, LogoIcon } from "@/components/layout/seashell-logo";

export default function HomePage() {
  const { hasHydrated, isAuthenticated } = useAuthStore();
  const showAuthenticated = hasHydrated && isAuthenticated;

  const { data: blocks, isLoading: blocksLoading } = useQuery({
    queryKey: ["blocks"],
    queryFn: publicApi.getBlocks,
    refetchInterval: 15_000,
  });

  const { data: values, isLoading: valuesLoading } = useQuery({
    queryKey: ["value"],
    queryFn: publicApi.getValue,
    refetchInterval: 30_000,
  });

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur">
        <div className="flex h-14 items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <LogoIcon size={28} className="rounded-full shadow-sm" />
            <span className="font-bold text-lg">SeaShell</span>
          </div>
          {showAuthenticated ? (
            <Button asChild size="sm">
              <Link href="/dashboard">
                <LayoutDashboard className="mr-2 h-4 w-4" />
                Dashboard
              </Link>
            </Button>
          ) : (
            <Button asChild size="sm">
              <Link href="/login">
                <LogIn className="mr-2 h-4 w-4" />
                Sign In
              </Link>
            </Button>
          )}
        </div>
      </header>

      <main className="px-4 py-6 space-y-6 max-w-2xl mx-auto">
        {/* Hero */}
        <div className="text-center py-6 flex flex-col items-center gap-4">
          <LogoWordmark width={220} height={165}  className="rounded-3xl shadow-md" />
          <p className="mt-1 text-muted-foreground text-sm">
            An cryptocurrency implementation in Go featuring a <strong>Proof of Authority</strong> blockchain, UTXO-based transactions, ECDSA wallets, a REST API with JWT authentication, and a multi-role management system.
          </p>
          {!showAuthenticated && (
            <div className="mt-4 flex gap-2 justify-center">
              <Button asChild>
                <Link href="/register">Get Started</Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/login">Sign In</Link>
              </Button>
            </div>
          )}
        </div>

        {/* Token Prices */}
        <section>
          <div className="flex items-center gap-2 mb-3">
            <TrendingUp className="h-4 w-4 text-primary" />
            <h2 className="font-semibold">Live Token Prices</h2>
          </div>
          {valuesLoading ? (
            <div className="space-y-2">
              {[1, 2].map((i) => <Skeleton key={i} className="h-16 w-full rounded-lg" />)}
            </div>
          ) : values && values.length > 0 ? (
            <div className="grid gap-2">
              {values.map((v) => (
                <Card key={v.authority_id}>
                  <CardContent className="flex items-center justify-between py-4 px-4">
                    <div>
                      <p className="font-medium text-sm">{v.authority_name}</p>
                      <p className="text-xs text-muted-foreground">Block #{v.block_height}</p>
                    </div>
                    <span className="text-lg font-bold text-primary">
                      {v.price.toFixed(4)} SHELL
                    </span>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">No active authorities yet.</p>
          )}
        </section>

        {/* Block Explorer */}
        <section>
          <div className="flex items-center gap-2 mb-3">
            <Layers className="h-4 w-4 text-primary" />
            <h2 className="font-semibold">Recent Blocks</h2>
          </div>
          {blocksLoading ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-20 w-full rounded-lg" />)}
            </div>
          ) : blocks && blocks.length > 0 ? (
            <div className="space-y-2">
              {blocks.map((block) => (
                <Card key={block.hash}>
                  <CardContent className="py-3 px-4">
                    <div className="flex items-start justify-between">
                      <div className="min-w-0">
                        <p className="font-mono text-xs text-muted-foreground truncate">
                          {truncateHash(block.hash, 10)}
                        </p>
                        <p className="text-sm font-medium mt-0.5">Block #{block.height}</p>
                        <p className="text-xs text-muted-foreground">{block.tx_count} txns · {formatRelative(block.timestamp)}</p>
                      </div>
                      <Badge variant={block.valid_poa ? "default" : "destructive"} className="shrink-0 ml-2">
                        {block.valid_poa ? "Valid" : "Invalid"}
                      </Badge>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">No blocks found. Initialize the chain first.</p>
          )}
        </section>
      </main>
    </div>
  );
}
