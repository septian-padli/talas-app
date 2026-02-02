

import { HomeIcon, BookmarkIcon, MagnifyingGlassIcon, BellIcon, UserIcon } from "@heroicons/react/24/outline"
import { cn } from "@/lib/utils"
import Link from "next/link"

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
                        <Icon className={cn("w-6 h-6", isActive ? "text-brand-400" : "text-white")} />
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