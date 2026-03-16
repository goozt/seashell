import type { Metadata, Viewport } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });
const jetbrainsMono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const metadata: Metadata = {
  title: "SeaShell",
  description: "Cryptocurrency ecosystem management",
  manifest: "/manifest.json",
  icons: {
    icon: [
      { url: "/ios/16.png",  sizes: "16x16",   type: "image/png" },
      { url: "/ios/32.png",  sizes: "32x32",   type: "image/png" },
      { url: "/ios/64.png",  sizes: "64x64",   type: "image/png" },
      { url: "/ios/128.png", sizes: "128x128", type: "image/png" },
      { url: "/ios/256.png", sizes: "256x256", type: "image/png" },
      { url: "/ios/512.png", sizes: "512x512", type: "image/png" },
    ],
    apple: [
      { url: "/ios/180.png",  sizes: "180x180"  },
      { url: "/ios/167.png",  sizes: "167x167"  },
      { url: "/ios/152.png",  sizes: "152x152"  },
      { url: "/ios/144.png",  sizes: "144x144"  },
      { url: "/ios/120.png",  sizes: "120x120"  },
      { url: "/ios/114.png",  sizes: "114x114"  },
      { url: "/ios/76.png",   sizes: "76x76"    },
      { url: "/ios/72.png",   sizes: "72x72"    },
      { url: "/ios/57.png",   sizes: "57x57"    },
    ],
  },
  appleWebApp: { capable: true, statusBarStyle: "default", title: "SeaShell" },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  themeColor: "#0ddff2",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta name="darkreader-lock" />
        {/* Windows tile */}
        <meta name="msapplication-TileColor" content="#a78bfa" />
        <meta name="msapplication-TileImage" content="/windows/Square150x150Logo.scale-100.png" />
        <meta name="msapplication-square70x70logo"   content="/windows/SmallTile.scale-100.png" />
        <meta name="msapplication-square150x150logo" content="/windows/Square150x150Logo.scale-100.png" />
        <meta name="msapplication-wide310x150logo"   content="/windows/Wide310x150Logo.scale-100.png" />
        <meta name="msapplication-square310x310logo" content="/windows/LargeTile.scale-100.png" />
      </head>
      <body className={`${inter.variable} ${jetbrainsMono.variable} font-display`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
