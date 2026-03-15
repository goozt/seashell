"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { archiveApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Archive, Download } from "lucide-react";

export default function ArchivePage() {
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [query, setQuery] = useState<{ from: number; to: number } | null>(null);

  const { data: stats } = useQuery({
    queryKey: ["archive-stats"],
    queryFn: archiveApi.getStats,
  });

  const { data: rangeData, isLoading: rangeLoading, error: rangeError } = useQuery({
    queryKey: ["archive-range", query],
    queryFn: () => archiveApi.getRange(query!.from, query!.to),
    enabled: !!query,
  });

  function handleQuery() {
    const f = parseInt(from, 10);
    const t = parseInt(to, 10);
    if (isNaN(f) || isNaN(t) || f > t || t - f > 1000) return;
    setQuery({ from: f, to: t });
  }

  function downloadJSON() {
    if (!rangeData?.blocks) return;
    const blob = new Blob([JSON.stringify(rangeData.blocks, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `archive_${query?.from}_${query?.to}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Archive className="h-5 w-5 text-primary" /> Block Archive
        </h1>
        <p className="text-sm text-muted-foreground">
          Query blocks that have been archived from the live chain.
        </p>
      </div>

      {stats && (
        <div className="grid grid-cols-3 gap-3">
          {[
            { label: "Total Archived", value: stats.total_blocks },
            { label: "Oldest Height", value: stats.oldest_height },
            { label: "Newest Height", value: stats.newest_height },
          ].map(({ label, value }) => (
            <Card key={label}>
              <CardContent className="py-3 text-center">
                <p className="text-2xl font-bold">{value}</p>
                <p className="text-xs text-muted-foreground">{label}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Query Range</CardTitle>
          <CardDescription>Retrieve up to 1000 archived blocks by height range.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-3 items-end">
            <div className="space-y-1 flex-1">
              <Label>From height</Label>
              <Input
                type="number"
                placeholder="0"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
              />
            </div>
            <div className="space-y-1 flex-1">
              <Label>To height</Label>
              <Input
                type="number"
                placeholder="99"
                value={to}
                onChange={(e) => setTo(e.target.value)}
              />
            </div>
            <Button onClick={handleQuery} disabled={rangeLoading}>
              {rangeLoading ? "Loading…" : "Query"}
            </Button>
          </div>

          {rangeError && (
            <Alert variant="destructive">
              <AlertDescription>{(rangeError as Error).message}</AlertDescription>
            </Alert>
          )}

          {rangeData && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">{rangeData.total} blocks returned</p>
                <Button variant="outline" size="sm" onClick={downloadJSON}>
                  <Download className="h-4 w-4 mr-2" /> Download JSON
                </Button>
              </div>
              <div className="divide-y max-h-96 overflow-y-auto rounded-md border">
                {rangeData.blocks.map((block) => (
                  <div key={block.hash} className="px-3 py-2 text-xs font-mono flex items-center justify-between gap-4">
                    <span className="text-muted-foreground w-16 shrink-0">#{block.height}</span>
                    <span className="truncate flex-1">{block.hash}</span>
                    <span className="text-muted-foreground shrink-0">
                      {new Date(block.timestamp * 1000).toLocaleDateString()}
                    </span>
                    <span className="text-muted-foreground shrink-0">
                      {block.signatures?.length ?? 0} sig{block.signatures?.length !== 1 ? "s" : ""}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
