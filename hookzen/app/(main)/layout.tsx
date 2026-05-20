"use client";

import Footer from "@/components/footer/Footer";
import Navbar from "@/components/navbar/Navbar";

interface MainLayoutProps {
  children: React.ReactNode;
}

function MainLayout({ children }: MainLayoutProps) {
  return (
    <div className="w-full flex flex-col items-center justify-center relative min-h-screen">
      <div className="w-full fixed top-0 z-50 backdrop-blur-xs flex items-center justify-center border-b border-zinc-900">
        <Navbar />
      </div>
      <div className="w-full">{children}</div>
      <div className="w-full">
        <Footer />
      </div>
    </div>
  );
}

export default MainLayout;
