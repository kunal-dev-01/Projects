import React from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "@/svg/Common";

interface BackButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  label?: string;
}

export default function BackButton({
  label = "Back",
  className = "",
  onClick,
  ...props
}: BackButtonProps) {
  const router = useRouter();

  const handleBack = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (onClick) {
      onClick(e);
    } else {
      router.back();
    }
  };

  return (
    <button
      onClick={handleBack}
      className={`group relative flex w-fit items-center gap-3 px-5 py-1 text-sm font-medium text-zinc-200 hover:text-white transition-colors duration-300 outline-none cursor-pointer border border-zinc-800 overflow-hidden ${className}`}
      {...props}
    >

      <div className="absolute inset-0 -z-10 bg-zinc-900 transition-transform duration-300 ease-out scale-x-0 origin-right group-hover:scale-x-100 group-hover:origin-left" />

      <ArrowLeft className="h-5 w-5 transition-transform duration-300 ease-out group-hover:-translate-x-1 text-white" />

      <span>{label}</span>
    </button>
  );
}