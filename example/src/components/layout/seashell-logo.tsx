import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/utils";

interface LogoIconProps {
  size?: number;
  className?: string;
}

/** Just the shell icon mark */
export function LogoIcon({ size = 32, className }: LogoIconProps) {
  return (
    <Image
      src="/logo-icon.png"
      alt="SeaShell"
      width={size}
      height={size}
      className={cn("shrink-0", className)}
      priority
    />
  );
}

interface LogoWordmarkProps {
  width?: number;
  height?: number;
  className?: string;
}

/** Shell icon + "SEASHELL" text wordmark */
export function LogoWordmark({ width = 200, height = 150, className }: LogoWordmarkProps) {
  return (
    <Image
      src="/logo-wordmark.png"
      alt="SeaShell"
      width={width}
      height={height}
      className={cn("shrink-0", className)}
      priority
    />
  );
}

interface LogoLinkProps {
  href?: string;
  iconSize?: number;
  className?: string;
}

/** Shell icon + brand name as a nav link */
export function LogoLink({ href = "/", iconSize = 28, className }: LogoLinkProps) {
  return (
    <Link href={href} className={cn("flex items-center gap-2", className)}>
      <LogoIcon size={iconSize} />
      <span className="font-bold text-base">SeaShell</span>
    </Link>
  );
}
