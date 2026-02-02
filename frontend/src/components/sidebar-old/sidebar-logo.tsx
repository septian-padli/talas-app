"use client";

import Image from "next/image";

interface SidebarLogoProps {
  isCollapsed: boolean;
}

export function SidebarLogo({ isCollapsed }: SidebarLogoProps) {
  return (
    <div className={isCollapsed ? "flex justify-center" : "px-6"}>
      <div className="flex items-center gap-x-3">
        <Image 
          src="/logo/talas-logo.png"
          alt="Talas Logo"
          width={35}
          height={35}
          className="object-contain"
        />
        {!isCollapsed && (
          <span className="text-xl font-medium text-white">
            talas
          </span>
        )}
      </div>
    </div>
  );
}
