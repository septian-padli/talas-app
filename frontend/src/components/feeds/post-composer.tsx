// components/home/molecules/post-composer.tsx
"use client";


import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { useUser } from "@/hooks/useUser";
import { UserProfile } from "@/types/auth";
import { AxiosError } from "axios";
import { AtSign, Image as ImageIcon } from "lucide-react";
import { cn } from "@/lib/utils";


export function PostComposer({ className = "" }: { className?: string }) {
  const router = useRouter();
  const { data: user, isLoading: userLoading, error: userError } = useUser() as { data?: UserProfile; isLoading: boolean; error: AxiosError | null; };

  const handleRedirectSearch = () => {
    router.push("/search");
  };
  const handleRedirectCreate = () => {
    router.push("/showcase/create");
  };


  // Use database name/username if available, otherwise use Clerk's data
  const currentUsername = user?.name || user?.username;

  return (
    <div className={cn("w-full", className)}>
      <div className="flex items-center gap-3 bg-[#181818] rounded-2xl px-4 py-3 ">
        {userLoading ? (
          <>
            <Skeleton className="w-10 h-10 rounded-full" />
            <div className="w-px h-10 bg-white/10 mx-4" />
            <div className="flex-1">
              <Skeleton className="h-10 w-full rounded-md" />
            </div>
            <Skeleton className="h-10 w-10 rounded-full ml-2" />
            <Skeleton className="h-10 w-10 rounded-full ml-2" />
            <Skeleton className="h-10 w-20 rounded-full ml-3" />
          </>
        ) : (
          <>
            <Avatar>
              <AvatarImage src={user?.avatar_url ?? undefined} alt={currentUsername} />
              <AvatarFallback>{currentUsername?.[0]?.toUpperCase() || 'U'}</AvatarFallback>
            </Avatar>
            <div className="w-px h-10 bg-white/10 mx-4" />
            <div className="flex-1">
              <input
                type="text"
                placeholder={`What's going on today, ${user?.name
                  ? user.name.split(" ").slice(0, 2).join(" ")
                  : ""
                  }?`}
                className="w-full bg-transparent border-none outline-none text-foreground placeholder:text-muted-foreground cursor-pointer text-base"
                onClick={handleRedirectSearch}
                readOnly
              />
            </div>
            {/* <button
              type="button"
              className="rounded-full p-2 hover:bg-white/10 transition-colors ml-2"
              tabIndex={-1}
              aria-label="Mention someone"
            >
              <AtSign className="w-5 h-5 text-white" />
            </button>
            <button
              type="button"
              className="rounded-full p-2 hover:bg-white/10 transition-colors ml-2"
              tabIndex={-1}
              aria-label="Add image"
            >
              <ImageIcon className="w-5 h-5 text-white" />
            </button> */}
            <Button
              onClick={handleRedirectCreate}
              // className="rounded-full px-6 bg-brand-400 text-white hover:bg-brand-500 ml-3"
              className="rounded-full ml-3 cursor-pointer"
              variant="brand"
              size="lg"
            >
              Create Showcase
            </Button>
          </>
        )}
      </div>
    </div>
  );
}