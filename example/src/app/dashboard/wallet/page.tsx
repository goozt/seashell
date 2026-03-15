"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Wallet, Copy, Check, Plus, RefreshCw } from "lucide-react";

export default function WalletPage() {
  const qc = useQueryClient();
  const [copied, setCopied] = useState(false);

  const { data: wallet, isLoading, error } = useQuery({
    queryKey: ["wallet"],
    queryFn: userApi.getWallet,
    retry: false,
  });

  const createMutation = useMutation({
    mutationFn: userApi.createWallet,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["wallet"] }),
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
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Wallet className="h-5 w-5 text-primary" /> Wallet
        </h1>
        <p className="text-sm text-muted-foreground">Manage your SHELL balance and address.</p>
      </div>

      {isLoading ? (
        <Card>
          <CardContent className="py-6 space-y-3">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-full" />
          </CardContent>
        </Card>
      ) : wallet ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Balance</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-4xl font-bold">
                {wallet.balance}
                <span className="text-lg font-normal text-muted-foreground ml-2">SHELL</span>
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Address</CardTitle>
              <CardDescription>Your public Base58 wallet address</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-2 p-3 rounded-md bg-muted">
                <p className="font-mono text-xs break-all flex-1">{wallet.address}</p>
                <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={copyAddress}>
                  {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => qc.invalidateQueries({ queryKey: ["wallet"] })}
                className="w-full"
              >
                <RefreshCw className="mr-2 h-4 w-4" /> Refresh Balance
              </Button>
            </CardContent>
          </Card>
        </>
      ) : (
        <Card>
          <CardContent className="py-8 text-center space-y-4">
            <Wallet className="h-12 w-12 mx-auto text-muted-foreground" />
            <div>
              <p className="font-semibold">No wallet yet</p>
              <p className="text-sm text-muted-foreground mt-1">
                Create a wallet to receive and send SHELL tokens.
              </p>
            </div>
            {createMutation.error && (
              <Alert variant="destructive">
                <AlertDescription>{(createMutation.error as Error).message}</AlertDescription>
              </Alert>
            )}
            <Button
              onClick={() => createMutation.mutate()}
              disabled={createMutation.isPending}
            >
              <Plus className="mr-2 h-4 w-4" />
              {createMutation.isPending ? "Creating…" : "Create Wallet"}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
