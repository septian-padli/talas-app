import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ArrowRightEndOnRectangleIcon,
  CogIcon,
} from "@heroicons/react/24/outline";
import {
  ArrowRightEndOnRectangleIcon as ArrowRightEndOnRectangleIconSolid,
  CogIcon as CogIconSolid,
} from "@heroicons/react/24/solid";
import LogoutButton from "../auth/LogoutButton";
import { cn } from "@/lib/utils";

interface MoreItem {
  icon: string;
  label: string;
  href: string;
  isLogout?: boolean;
}

interface MoreLeftSidebarProps {
  isCollapsed: boolean;
  activeItem?: string;
  isMobile?: boolean;
}

export function MoreLeftSidebar({
  isCollapsed,
  activeItem,
  isMobile = false,
}: MoreLeftSidebarProps) {
  const pathname = usePathname();

  const moreItems: MoreItem[] = [
    { icon: "settings", label: "Settings", href: "/settings" },
    { icon: "logout", label: "Log Out", href: "#", isLogout: true },
  ];

  const renderIcon = (item: MoreItem, isActive: boolean) => {
    if (item.icon === "settings") {
      return isActive ? (
        <CogIconSolid className="w-6 h-6" />
      ) : (
        <CogIcon className="w-6 h-6" />
      );
    }
    if (item.icon === "logout") {
      return isActive ? (
        <ArrowRightEndOnRectangleIconSolid className="w-6 h-6 text-[#EF4444]" />
      ) : (
        <ArrowRightEndOnRectangleIcon className="w-6 h-6 text-[#EF4444]" />
      );
    }
    return null;
  };

  if (isMobile) {
    return (
      <nav className="w-full border">
        <ul className="flex justify-evenly w-full">
          {moreItems.map((item) => {
            const isActive = item.label.toLowerCase() === activeItem?.toLowerCase();

            return (
              <li key={item.label}>
                {item.isLogout ? (
                  <LogoutButton variant={"ghost"} />
                ) : (
                  <Link
                    href={item.href}
                    className={`flex items-center justify-center p-2 rounded-md transition-colors ${isActive ? "text-primary" : "text-white hover:bg-white/10"
                      }`}
                    title={item.label}
                  >
                    {renderIcon(item, isActive)}
                  </Link>
                )}
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
        {moreItems.map((item) => {
          const isActive = item.label.toLowerCase() === activeItem?.toLowerCase();

          return (
            <li key={item.label}>
              {item.isLogout ? (
                <LogoutButton
                  variant={"destructive"}
                  size={isCollapsed ? "icon" : "lg"}
                  className={cn(
                    isCollapsed ? "justify-center" : "w-full",
                    "transition-tranformation duration-200 active:scale-90"
                  )}
                />
              ) : (
                <Link
                  href={item.href}
                  className={`flex items-center ${isCollapsed ? "justify-center" : ""
                    } gap-3 w-full px-6 py-4 text-base font-medium transition-tranformation duration-200 active:scale-90 ${isActive ? "text-primary" : "text-white hover:bg-white/10"
                    } rounded-md`}
                  title={isCollapsed ? item.label : ""}
                >
                  {renderIcon(item, isActive)}
                  {!isCollapsed && <span>{item.label}</span>}
                </Link>
              )}
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
