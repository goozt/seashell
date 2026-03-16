"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { publicApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { truncateHash, formatRelative } from "@/lib/utils";
import {
  TrendingUp,
  Layers,
  LogIn,
  LayoutDashboard,
  ShieldCheck,
  UserCheck,
  Gavel,
} from "lucide-react";
import { LogoIcon, LogoWordmark } from "@/components/layout/seashell-logo";

function RecentEvents() {
  const { data: events, isLoading } = useQuery({
    queryKey: ["chain-events"],
    queryFn: () => publicApi.getChainEvents(),
    refetchInterval: 30_000,
  });

  if (isLoading)
    return (
      <section>
        <div className="flex items-center gap-2 mb-3">
          <Gavel className="h-4 w-4 text-primary" />
          <h2 className="text-base font-bold">Governance Feed</h2>
        </div>
        <Skeleton className="h-24 w-full rounded-xl" />
      </section>
    );
  if (!events || events.length === 0) return null;

  return (
    <section>
      <div className="flex items-center gap-2 mb-4">
        <Gavel className="h-4 w-4 text-primary" />
        <h2 className="text-base font-bold">Governance Feed</h2>
      </div>
      {/* Timeline */}
      <div className="relative space-y-4 before:absolute before:inset-y-0 before:left-4 before:w-0.5 before:bg-primary/10">
        {events.slice(0, 10).map((e, i) => (
          <div key={i} className="relative pl-10">
            <div
              className={`absolute left-0 top-1 size-8 rounded-full flex items-center justify-center border ${
                e.type === "authority_registered"
                  ? "bg-amber-400/20 text-amber-400 border-amber-400/30"
                  : "bg-primary/20 text-primary border-primary/30"
              }`}
            >
              {e.type === "authority_registered" ? (
                <ShieldCheck className="h-4 w-4" />
              ) : (
                <UserCheck className="h-4 w-4" />
              )}
            </div>
            <div className="glass-card dark:glass-card glass-card-light p-4 rounded-xl">
              <h4 className="text-sm font-bold">
                {e.type === "authority_registered"
                  ? "Authority Registered"
                  : "User Verified"}
              </h4>
              <p className="text-xs text-muted-foreground mt-1">
                {e.type === "authority_registered" ? (
                  <>
                    <span className="text-amber-500 dark:text-amber-400 font-medium">
                      {e.authority_name}
                    </span>{" "}
                    joined the network as a validator authority.
                  </>
                ) : (
                  <>
                    Identity{" "}
                    <span className="font-mono text-primary">
                      {e.user_id_hash ? `${e.user_id_hash.slice(0, 6)}…${e.user_id_hash.slice(-4)}` : "—"}
                    </span>{" "}
                    verified by {e.authority_name}.
                  </>
                )}
              </p>
              <span className="text-[10px] text-muted-foreground mt-2 block">
                Block #{e.height}
              </span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

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
      <header className="sticky top-0 z-40 bg-background/80 backdrop-blur-md border-b border-primary/10">
        <div className="flex h-14 items-center justify-between px-4 max-w-2xl mx-auto">
          <div className="flex items-center gap-2">
            <LogoIcon size={28} className="rounded-full" />
            <span className="font-bold text-base">SeaShell{" "}
              <span className="text-primary/80 font-medium">Public</span>
            </span>
          </div>
          {showAuthenticated ? (
            <Button asChild size="sm" variant="outline" className="border-primary/30 text-primary hover:bg-primary/10">
              <Link href="/dashboard">
                <LayoutDashboard className="mr-2 h-4 w-4" />
                Dashboard
              </Link>
            </Button>
          ) : (
            <Button asChild size="sm" className="bg-primary text-primary-foreground hover:bg-primary/90">
              <Link href="/login">
                <LogIn className="mr-2 h-4 w-4" />
                Sign In
              </Link>
            </Button>
          )}
        </div>
      </header>

      <main className="px-4 py-6 space-y-8 max-w-2xl mx-auto pb-12">
        {/* Hero */}
        <div className="text-center py-6 flex flex-col items-center gap-4">
          <LogoWordmark width={200} height={150} className="rounded-3xl shadow-md" />
          <p className="text-sm text-muted-foreground max-w-xs">
            A{" "}
            <strong className="text-foreground">Proof of Authority</strong>{" "}
            blockchain network for federated financial ecosystems.
          </p>
          {!showAuthenticated && (
            <div className="flex gap-2 justify-center mt-2">
              <Button asChild className="bg-primary text-primary-foreground hover:bg-primary/90">
                <Link href="/register">Get Started</Link>
              </Button>
              <Button asChild variant="outline" className="border-primary/30 text-primary hover:bg-primary/10">
                <Link href="/login">Sign In</Link>
              </Button>
            </div>
          )}
        </div>

        {/* Live Token Prices */}
        <section>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-primary" />
              <h2 className="text-base font-bold">Live Token Prices</h2>
            </div>
            <span className="text-[10px] font-bold text-primary animate-pulse uppercase tracking-wider">Live</span>
          </div>
          {valuesLoading ? (
            <div className="grid grid-cols-2 gap-3">
              {[1, 2].map((i) => <Skeleton key={i} className="h-20 w-full rounded-xl" />)}
            </div>
          ) : values && values.length > 0 ? (
            <div className="grid grid-cols-2 gap-3">
              {values.map((v) => (
                <div key={v.authority_id} className="glass-card dark:glass-card glass-card-light p-4 rounded-xl flex flex-col gap-2">
                  <div className="flex items-center justify-between">
                    <div className="bg-primary/10 p-1.5 rounded-lg">
                      <TrendingUp className="h-4 w-4 text-primary" />
                    </div>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground truncate">{v.authority_name}</p>
                    <p className="text-lg font-bold font-mono">
                      {v.price.toFixed(4)}{" "}
                      <span className="text-xs font-normal text-muted-foreground">SHELL</span>
                    </p>
                    {v.block_height > 0 && (
                      <p className="text-[10px] text-muted-foreground">Block #{v.block_height}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">No active authorities yet.</p>
          )}
        </section>

        {/* Recent Blocks */}
        <section>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Layers className="h-4 w-4 text-primary" />
              <h2 className="text-base font-bold">Recent Blocks</h2>
            </div>
          </div>
          {blocksLoading ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)}
            </div>
          ) : blocks && blocks.length > 0 ? (
            <div className="space-y-2">
              {blocks.map((block, idx) => (
                <div
                  key={block.hash}
                  className={`glass-card dark:glass-card glass-card-light p-4 rounded-xl flex items-center justify-between border-l-4 ${
                    idx === 0 ? "border-l-primary" : "border-l-primary/20"
                  }`}
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-muted-foreground">Block #{block.height}</p>
                    <p className="font-mono text-sm truncate">{truncateHash(block.hash, 10)}</p>
                    {block.event_count && block.event_count > 0 ? (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {block.events?.map((e, i) => (
                          <Badge key={i} variant="outline" className="text-[10px] gap-1 py-0 border-primary/20 text-primary">
                            {e.type === "authority_registered"
                              ? <><ShieldCheck className="h-2.5 w-2.5" />{e.authority_name}</>
                              : <><UserCheck className="h-2.5 w-2.5" />Verified</>}
                          </Badge>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  <div className="text-right ml-4 shrink-0">
                    <p className="text-xs font-medium">{block.tx_count} TXs</p>
                    <p className="text-[10px] text-muted-foreground">{formatRelative(block.timestamp)}</p>
                    <Badge
                      variant={block.valid_poa ? "default" : "destructive"}
                      className="text-[10px] mt-1"
                    >
                      {block.valid_poa ? "Valid" : "Invalid"}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">No blocks yet.</p>
          )}
        </section>

        <RecentEvents />
      </main>
    </div>
  );
}
