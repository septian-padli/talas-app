import type { Metadata } from "next";
import { Comfortaa, Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import QueryProvider from "@/providers/QueryProvider";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const comfortaa = Comfortaa({
  variable: "--font-comfortaa",
  subsets: ["latin"],
  weight: ["400", "700"],
})

export const metadata: Metadata = {
  title: "Talas - Showcase Your Projects",
  description: "A  web-based social media platform designed to showcase software engineering projects. Talas enables users to share, explore, and interact with innovative project portfolios in a modern and engaging way. Built for collaboration, inspiration, and growth in the tech community.",
  icons: {
    icon: "/logo/talas-logo.png",
    shortcut: "/logo/talas-logo.png",
    apple: "/logo/talas-logo.png",
  },
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      {/* head html */}
      <head>
        <link
          rel="icon"
          href="/logo/talas-logo.png"
          sizes="48x48"
          type="image/png"
        />
        <link rel="apple-touch-icon" href="/logo/talas-logo.png" />
      </head>
      <body
        className={`${comfortaa.variable} font-sans dark antialiased`}
      >
        <QueryProvider>
          {children}
        </QueryProvider>
      </body>
    </html>
  );
}
