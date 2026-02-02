"use client";


import * as React from "react";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import type * as ReactType from "react";
import { LogOut, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { authService } from "@/services/authService";
import {
    AlertDialog,
    AlertDialogTrigger,
    AlertDialogContent,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogCancel,
    AlertDialogAction,
} from "@/components/ui/alert-dialog";


export interface LogoutButtonProps extends ReactType.ComponentProps<typeof Button> {
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
    const [open, setOpen] = useState(false);
    const mutation = useMutation({
        mutationFn: async () => {
            await authService.logout();
        },
        onSuccess: () => {
            setOpen(false);
            router.replace(redirectTo);
        },
        onError: (err) => {
            setOpen(false);
            console.error("Logout failed", err);
            router.replace(redirectTo);
        },
    });

    // Handler for the destructive action button
    const handleLogout = (e: React.MouseEvent<HTMLButtonElement>) => {
        // Prevent modal from closing automatically while loading
        if (mutation.isPending) {
            e.preventDefault();
            return;
        }
        mutation.mutate();
    };

    return (
        <AlertDialog open={open} onOpenChange={setOpen}>
            <AlertDialogTrigger asChild>
                <Button
                    type="button"
                    variant={props.variant || "ghost"}
                    size={iconOnly ? "icon" : props.size}
                    className={cn(className)}
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
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Are you sure you want to log out?</AlertDialogTitle>
                    <AlertDialogDescription>
                        You will need to log in again to access your account.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={mutation.isPending}>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        variant="destructive"
                        onClick={handleLogout}
                        disabled={mutation.isPending}
                    >
                        {mutation.isPending ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                            <LogOut className="mr-2 h-4 w-4" />
                        )}
                        Yes, Log out
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

export default LogoutButton;
