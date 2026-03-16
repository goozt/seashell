"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { cn } from "@/lib/utils";
import {
  Home,
  Wallet,
  ArrowLeftRight,
  Building2,
  MessageSquareText,
  ShieldCheck,
  Users,
  FileQuestion,
  Star,
  Network,
  TicketCheck,
} from "lucide-react";

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
}

const navItems: NavItem[] = [
  { label: "Home",      href: "/dashboard",              icon: Home },
  { label: "Wallet",    href: "/dashboard/wallet",       icon: Wallet },
  { label: "Send",      href: "/dashboard/transactions", icon: ArrowLeftRight },
  { label: "Authority", href: "/dashboard/authority",    icon: Building2 },
  { label: "Support",   href: "/dashboard/tickets",      icon: MessageSquareText },
];

const adminItems: NavItem[] = [
  { label: "Requests", href: "/admin/requests", icon: FileQuestion },
  { label: "Support",  href: "/admin/tickets",  icon: TicketCheck },
  { label: "Users",    href: "/admin/users",    icon: Users },
  { label: "Network",  href: "/admin/network",  icon: Network },
  { label: "Stats",    href: "/admin",          icon: ShieldCheck },
];

const superAdminItems: NavItem[] = [
  { label: "Dashboard", href: "/superadmin",         icon: Star },
  { label: "Admins",    href: "/superadmin/admins",  icon: ShieldCheck },
  { label: "Support",   href: "/superadmin/tickets", icon: TicketCheck },
];

export function MobileNav() {
  const pathname = usePathname();
  const { hasHydrated, user } = useAuthStore();

  if (!hasHydrated) return null;

  const items =
    user?.role === "superadmin"
      ? superAdminItems
      : user?.role === "admin"
      ? adminItems
      : navItems;

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-background/90 backdrop-blur-xl border-t border-primary/10 pb-safe">
      <div className="flex h-16 max-w-2xl mx-auto">
        {items.map(({ label, href, icon: Icon }) => {
          const active =
            pathname === href ||
            (href !== "/dashboard" && pathname.startsWith(href));
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex flex-1 flex-col items-center justify-center gap-1 text-[10px] font-bold uppercase tracking-wider transition-colors",
                active ? "text-primary" : "text-muted-foreground hover:text-foreground"
              )}
            >
              <div
                className={cn(
                  "flex items-center justify-center h-8 w-8 rounded-xl transition-colors",
                  active && "bg-primary/10"
                )}
              >
                <Icon className="h-5 w-5" />
              </div>
              <span>{label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
