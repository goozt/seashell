"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

// Redirect to main superadmin page which has the admins tab
export default function AdminsRedirect() {
  const router = useRouter();
  useEffect(() => { router.replace("/superadmin"); }, [router]);
  return null;
}
