"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils"; // Ensure cn is imported
import {
  HomeIcon,
  BookmarkIcon,
  MagnifyingGlassIcon,
  BellIcon,
  UserIcon
} from "@heroicons/react/24/outline";
import {
  HomeIcon as HomeIconSolid,
  BookmarkIcon as BookmarkIconSolid,
  MagnifyingGlassIcon as MagnifyingGlassIconSolid,
  BellIcon as BellIconSolid,
  UserIcon as UserIconSolid
} from "@heroicons/react/24/solid";
import { useUser } from "@/hooks/useUser";

interface NavItem {
  icon: string;
  label: string;
  href: string;
}

interface SidebarNavProps {
  isCollapsed: boolean;
  activeItem?: string;
  isMobile?: boolean;
}

export function SidebarNav({ isCollapsed, activeItem, isMobile = false }: SidebarNavProps) {
  const pathname = usePathname();
  const { data: user, isLoading: isUserLoading, isError: isUserError } = useUser();
  const userId = user?.id;
  const username = user?.username;

  const navItems: NavItem[] = [
    { icon: "home", label: "Home", href: "/" },
    { icon: "bookmark", label: "Saved projects", href: "/saved" },
    { icon: "search", label: "Search", href: "/search" },
    { icon: "bell", label: "Notification", href: "/notifications" },
    {
      icon: "user",
      label: "Profile",
      href: username ? `/profile/${username}` : "/", // fallback ke "/" kalau belum ada username
    },
  ];

  if (isMobile) {
    return (
      <nav className="w-full">
        <ul className="flex justify-evenly w-full">
          {navItems.map((item) => {
            const isActive = activeItem
              ? item.label.toLowerCase() === activeItem.toLowerCase()
              : pathname === item.href;

            return (
              <li key={item.label}>
                <Link
                  href={item.href}
                  className={cn(
                    "flex items-center justify-center p-2 rounded-md",
                    "transition-all duration-200 ease-in-out", // Added for smooth transitions
                    "hover:scale-105 active:scale-90", // Scale animations
                    isActive ? "text-primary" : "text-white hover:bg-white/10"
                  )}
                  title={item.label}
                >
                  {/* Icon rendering logic remains the same */}
                  {item.icon === "home" && (isActive ? <HomeIconSolid className="w-6 h-6" /> : <HomeIcon className="w-6 h-6" />)}
                  {item.icon === "bookmark" && (isActive ? <BookmarkIconSolid className="w-6 h-6" /> : <BookmarkIcon className="w-6 h-6" />)}
                  {item.icon === "search" && (isActive ? <MagnifyingGlassIconSolid className="w-6 h-6" /> : <MagnifyingGlassIcon className="w-6 h-6" />)}
                  {item.icon === "bell" && (
                    isActive ? (
                      <BellIconSolid className="w-6 h-6" />
                    ) : (
                      <span className="relative">
                        <BellIcon className="w-6 h-6" />
                        {/* {isUnRead.data?.data ? (
                          <span className="absolute top-0 right-0 block w-2.5 h-2.5 bg-red-500 rounded-full" />
                        ) : ""} */}
                      </span>
                    )
                  )}
                  {item.icon === "user" && (isActive ? <UserIconSolid className="w-6 h-6" /> : <UserIcon className="w-6 h-6" />)}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    );
  }

  return (
    <nav>
      <ul className="flex flex-col gap-y-2">
        {navItems.map((item) => {
          const isActive = activeItem
            ? item.label.toLowerCase() === activeItem.toLowerCase()
            : pathname === item.href;

          return (
            <li key={item.label}>
              <Link
                href={item.href}
                className={cn(
                  "flex items-center gap-3 w-full px-6 py-4 text-base font-medium rounded-md",
                  isCollapsed ? "justify-center" : "",
                  "transition-all duration-200 ease-in-out", // General transition for all properties
                  "hover:scale-105 active:scale-90", // Scale animations
                  isActive ? "text-primary" : "text-white hover:bg-white/10" // Adjusted active state for better visibility
                )}
                title={isCollapsed ? item.label : ""}
              >
                {/* Icon rendering logic remains the same */}
                {item.icon === "home" && (isActive ? <HomeIconSolid className="w-6 h-6" /> : <HomeIcon className="w-6 h-6" />)}
                {item.icon === "bookmark" && (isActive ? <BookmarkIconSolid className="w-6 h-6" /> : <BookmarkIcon className="w-6 h-6" />)}
                {item.icon === "search" && (isActive ? <MagnifyingGlassIconSolid className="w-6 h-6" /> : <MagnifyingGlassIcon className="w-6 h-6" />)}
                {item.icon === "bell" && (
                  isActive ? (
                    <BellIconSolid className="w-6 h-6" />
                  ) : (
                    <span className="relative">
                      <BellIcon className="w-6 h-6" />
                      {/* {isUnRead.data?.data ? (
                        <span className="absolute top-0 right-0 block w-2.5 h-2.5 bg-red-500 rounded-full" />
                      ) : ""} */}
                    </span>
                  )
                )}
                {item.icon === "user" && (isActive ? <UserIconSolid className="w-6 h-6" /> : <UserIcon className="w-6 h-6" />)}

                {!isCollapsed && <span>{item.label}</span>}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
