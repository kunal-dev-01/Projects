import type { Metadata } from "next";
import "./globals.css";
import Provider from "./provider";


export const metadata: Metadata = {
  title: "Hookzen",
  description: "",
  icons: {
    icon: "/favicon.svg"
  }
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`h-full antialiased`}
      cz-shortcut-listen="true"
    >
      <body className="min-h-full flex flex-col">
        <Provider>
          {children}
        </Provider>
      </body>
    </html>
  );
}
