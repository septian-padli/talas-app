import React from "react";
import { Skeleton } from "@/components/ui/skeleton";

export function NotificationItemSkeleton() {
  return (
    <div className="px-4 py-3 flex items-start transition-all border-b border-gray-700 last:border-b-0 opacity-70">
      <div className="shrink-0 mt-1 mr-4">
        <Skeleton className="w-8 h-8 rounded-full" />
      </div>
      <div className="flex-1">
        <Skeleton className="h-4 w-3/4 mb-2" />
        <Skeleton className="h-3 w-1/2" />
      </div>
      <div className="text-xs text-gray-400 ml-2">
        <Skeleton className="h-3 w-8" />
      </div>
    </div>
  );
}
