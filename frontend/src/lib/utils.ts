import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date) {
  return new Intl.DateTimeFormat("pt-BR", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(date))
}

export function formatNumber(num: number) {
  return new Intl.NumberFormat("pt-BR").format(num)
}

export function getStatusColor(status: string) {
  const statusColors = {
    pending: "text-yellow-600 bg-yellow-100",
    sent: "text-green-600 bg-green-100", 
    failed: "text-red-600 bg-red-100",
    retrying: "text-orange-600 bg-orange-100",
    accepted: "text-green-600 bg-green-100",
    expired: "text-red-600 bg-red-100",
    revoked: "text-gray-600 bg-gray-100",
  };
  return statusColors[status as keyof typeof statusColors] || "text-gray-600 bg-gray-100";
}