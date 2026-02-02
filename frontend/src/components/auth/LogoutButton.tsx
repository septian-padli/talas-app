"use client";

import * as React from "react";
import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import type * as ReactType from "react";

type ButtonProps = ReactType.ComponentProps<typeof Button>;
import { LogOut, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { authService } from "@/services/authService";

export interface LogoutButtonProps extends ButtonProps {
    redirectTo?: string;
    iconOnly?: boolean;
}

export function LogoutButton({
    redirectTo = "/login",
    iconOnly = false,
    className,
    ...props
}: LogoutButtonProps) {
    const router = useRouter();
    const mutation = useMutation({
        mutationFn: async () => {
            // You may want to call an API endpoint here if needed
            await authService.logout();
        },
        onSuccess: () => {
            router.replace(redirectTo);
        },
        onError: (err) => {
            // Fallback: force redirect/cleanup
            console.error("Logout failed", err);
            router.replace(redirectTo);
        },
    });

    return (
        <Button
            type="button"
            variant={props.variant || "ghost"}
            size={iconOnly ? "icon" : props.size}
            className={cn(className)}
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending || props.disabled}
            aria-label="Log out"
            {...props}
        >
            {mutation.isPending ? (
                <Loader2 className={cn("mr-2 h-4 w-4 animate-spin", iconOnly && "m-0")} />
            ) : (
                <LogOut className={cn("mr-2 h-4 w-4", iconOnly && "m-0")} />
            )}
            {!iconOnly && "Log out"}
        </Button>
    );
}

export default LogoutButton;
