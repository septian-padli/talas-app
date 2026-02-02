"use client";

import { useEffect, useState } from "react";
import { SidebarLogo } from "./sidebar-logo";
import { SidebarNav } from "./sidebar-nav";
import { SidebarMore } from "./sidebar-more";
import LogoutButton from "../auth/LogoutButton";

interface SidebarProps {
  activeItem?: string;
}

export function SidebarOld({ activeItem }: SidebarProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    // Function to check window width and update sidebar state
    const handleResize = () => {
      const width = window.innerWidth;
      setIsCollapsed(width <= 1180);
      setIsMobile(width <= 690);
    };

    // Set initial state
    handleResize();

    // Add event listener for window resize
    window.addEventListener('resize', handleResize);

    // Clean up event listener on component unmount
    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  if (isMobile) {
    return (
      <>
        {/* Top bar for mobile - simplified with just a border */}
        <div className="fixed top-0 left-0 right-0 h-16 bg-background flex items-center justify-between px-4 z-50 border-b-2 border-white/10">
          <div className="flex-1"></div>
          <div className="flex justify-center">
            <SidebarLogo isCollapsed={true} />
          </div>
          <div className="flex-1 flex justify-end">
            <SidebarMore isCollapsed={true} />
          </div>
        </div>

        {/* Bottom navigation for mobile - simplified with just a border */}
        <div className="fixed bottom-0 left-0 right-0 h-16 bg-background flex items-center justify-evenly z-50 border-t-2 border-white/10">
          <SidebarNav isCollapsed={true} activeItem={activeItem} isMobile={true} />
        </div>
      </>
    );
  }

  return (
    <aside
      className={`h-screen bg-background fixed left-0 top-0 flex flex-col justify-between py-7 ${isCollapsed ? "w-[70px] items-center px-0" : "w-[270px] px-4"
        }`}
    >
      {/* Logo section */}
      <SidebarLogo isCollapsed={isCollapsed} />

      {/* Navigation section */}
      <SidebarNav isCollapsed={isCollapsed} activeItem={activeItem} />

      {/* More button section */}
      <LogoutButton variant={"outline"} size={"lg"} />
      {/* <SidebarMore isCollapsed={isCollapsed} /> */}
    </aside>
  );
}