

import { HomeIcon, BookmarkIcon, MagnifyingGlassIcon, BellIcon, UserIcon } from "@heroicons/react/24/outline"
import { cn } from "@/lib/utils"
import Link from "next/link"
import { useNotifications } from "@/hooks/useNotifications"

const navItems = [
    { icon: HomeIcon, label: "Home", url: "/" },
    { icon: BookmarkIcon, label: "Bookmark", url: "/bookmark" },
    { icon: MagnifyingGlassIcon, label: "Search", url: "/search" },
    { icon: BellIcon, label: "Notification", url: "/notification" },
    { icon: UserIcon, label: "Profile", url: "/profile/me" },
];

interface SideNavProps {
    activeItem?: string;
    orientation?: "vertical" | "horizontal";
}

const SideNav: React.FC<SideNavProps> = ({ activeItem, orientation = "vertical" }) => {

    const { unreadCount } = useNotifications();
    return (
        <nav className={cn(
            orientation === "horizontal"
                ? "flex flex-row w-full h-full justify-evenly items-center"
                : "flex flex-col w-full"
        )}>
            {navItems.map((item) => {
                const isActive = item.label === activeItem;
                const Icon = item.icon;
                return (
                    <Link
                        href={item.url}
                        key={item.label}
                        className={cn(
                            orientation === "horizontal"
                                ? "flex flex-col items-center justify-center flex-1 py-2"
                                : "flex items-center gap-4 text-lg font-medium cursor-pointer hover:bg-white/10 rounded-md px-6 py-4 transition-colors duration-150",
                            isActive ? "text-brand-400" : "text-white"
                        )}
                    >
                        <div className="relative">
                            <Icon className={cn("w-6 h-6", isActive ? "text-brand-400" : "text-white")} />
                            {item.label === "Notification" && unreadCount > 0 && (
                                <span className="absolute -top-1 -right-1 min-w-4.5 h-5 px-1 flex items-center justify-center text-xs font-bold bg-red-500 text-white rounded-full border-2 border-background z-10">
                                    {unreadCount > 99 ? '99+' : unreadCount}
                                </span>
                            )}
                        </div>
                        {orientation !== "horizontal" && (
                            <span>{item.label}</span>
                        )}
                    </Link>
                );
            })}
        </nav>
    );
}

export default SideNav