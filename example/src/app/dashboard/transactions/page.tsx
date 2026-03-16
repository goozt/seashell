"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { ArrowLeftRight, Send, Loader2, ShieldCheck } from "lucide-react";
import { truncateHash } from "@/lib/utils";

export default function TransactionsPage() {
  const qc = useQueryClient();
  const [form, setForm] = useState({ to_address: "", amount: "" });
  const [success, setSuccess] = useState("");
  const [error, setError] = useState("");

  const { data: transactions, isLoading } = useQuery({
    queryKey: ["transactions"],
    queryFn: userApi.getTransactions,
    retry: false,
  });

  const sendMutation = useMutation({
    mutationFn: (body: { to_address: string; amount: number }) =>
      userApi.sendTransaction(body),
    onSuccess: () => {
      setSuccess("Transaction submitted successfully!");
      setForm({ to_address: "", amount: "" });
      qc.invalidateQueries({ queryKey: ["transactions"] });
      qc.invalidateQueries({ queryKey: ["wallet"] });
      setTimeout(() => setSuccess(""), 5000);
    },
    onError: (err: Error) => setError(err.message),
  });

  function handleSend(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess("");
    const amount = parseInt(form.amount, 10);
    if (!amount || amount <= 0) {
      setError("Amount must be a positive integer.");
      return;
    }
    sendMutation.mutate({ to_address: form.to_address, amount });
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <ArrowLeftRight className="h-5 w-5 text-primary" /> Transactions
        </h1>
        <p className="text-sm text-muted-foreground">Send SHELL and view history.</p>
      </div>

      {/* Send form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Send className="h-4 w-4" /> Send SHELL
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSend} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            {success && (
              <Alert>
                <AlertDescription className="text-green-600">{success}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="to_address">Recipient Address</Label>
              <Input
                id="to_address"
                placeholder="Base58 address…"
                value={form.to_address}
                onChange={(e) => setForm((f) => ({ ...f, to_address: e.target.value }))}
                required
                className="font-mono text-sm"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="amount">Amount (SHELL)</Label>
              <Input
                id="amount"
                type="number"
                min={1}
                step={1}
                placeholder="10"
                value={form.amount}
                onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
                required
              />
            </div>
            <Button type="submit" className="w-full" disabled={sendMutation.isPending}>
              {sendMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Send
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* History */}
      <div>
        <h2 className="text-sm font-semibold mb-2">Transaction History</h2>
        {isLoading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 rounded-lg" />)}
          </div>
        ) : transactions && transactions.length > 0 ? (
          <div className="space-y-2">
            {transactions.map((tx, idx) => {
              const txId = tx.tx_id ?? tx.id;
              const verified = !!tx.identity_proof;
              return (
                <Card key={txId ?? idx}>
                  <CardContent className="py-3 px-4">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        {txId && (
                          <p className="font-mono text-xs text-muted-foreground truncate">
                            {truncateHash(txId, 8)}
                          </p>
                        )}
                        {tx.to_address && (
                          <p className="text-xs text-muted-foreground mt-0.5">
                            To: <span className="font-mono">{truncateHash(tx.to_address, 8)}</span>
                          </p>
                        )}
                        {tx.block_height && (
                          <p className="text-xs text-muted-foreground">Block #{tx.block_height}</p>
                        )}
                        {verified && (
                          <p className="text-xs text-muted-foreground mt-0.5">
                            Verified by: <span className="font-medium">{tx.identity_proof!.authority_name}</span>
                          </p>
                        )}
                      </div>
                      <div className="flex flex-col items-end gap-1 shrink-0">
                        {tx.amount !== undefined && (
                          <Badge variant="outline">{tx.amount} SHELL</Badge>
                        )}
                        {verified && (
                          <Badge variant="secondary" className="flex items-center gap-1 text-green-700 bg-green-50 border-green-200">
                            <ShieldCheck className="h-3 w-3" /> Verified
                          </Badge>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground text-center py-6">No transactions yet.</p>
        )}
      </div>
    </div>
  );
}
