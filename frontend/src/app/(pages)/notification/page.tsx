"use client";
import { PostCard } from "@/components/feeds/post-card";
import { PostComposer } from "@/components/feeds/post-composer";
import { NotificationItems } from "@/components/notification/notifItem";
import { NotificationItemSkeleton } from "@/components/notification/notifItemSkeleton";
import { useNotifications } from "@/hooks/useNotifications";
import { useEffect, useRef } from "react";

interface NotificationPageProps {
    prop: string;
}

const NotificationPage: React.FC<NotificationPageProps> = () => {
    const { notifications, isLoading, unreadCount, markAllRead } = useNotifications();
    const hasMarkedRef = useRef(false);

    useEffect(() => {
        if (!isLoading && unreadCount > 0 && !hasMarkedRef.current) {
            markAllRead();
            hasMarkedRef.current = true; // Kunci agar tidak nembak lagi
        }
    }, [isLoading, unreadCount, markAllRead]);

    return (
        <div className="bg-[#181818] rounded-2xl">
            <div className="flex justify-center">
                <div className="bg-[#1a1a1a] rounded-xl border border-white/10 p-6 w-full ">
                    <div className="flex-1 overflow-y-auto">
                        <section className="space-y-4 mb-6">
                            <h2 className="text-sm text-gray-400 font-medium mb-2">Recent</h2>
                            {isLoading
                                ? Array.from({ length: 3 }).map((_, idx) => (
                                    <NotificationItemSkeleton key={idx} />
                                ))
                                : notifications.map((notification) => (
                                    <NotificationItems
                                        key={notification.id}
                                        notification={notification}
                                    />
                                ))}
                        </section>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default NotificationPage;
