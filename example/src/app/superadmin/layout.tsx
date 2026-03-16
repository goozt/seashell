import { Header } from "@/components/layout/header";
import { MobileNav } from "@/components/layout/mobile-nav";
import { SuperAdminAuthGuard } from "@/components/layout/superadmin-auth-guard";

export default function SuperAdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col min-h-screen">
      <Header />
      <SuperAdminAuthGuard>
        <main className="flex-1 px-4 py-4 pb-24 max-w-2xl mx-auto w-full">
          {children}
        </main>
      </SuperAdminAuthGuard>
      <MobileNav />
    </div>
  );
}
