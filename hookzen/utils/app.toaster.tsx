"use client";

import { Toaster } from "sonner";
import { useEffect, useState } from "react";

export default function AppToaster() {
    const [muted,setMuted] = useState(false)

    useEffect(() => {
        setMuted(true)
    },[])

    return (
        <Toaster position="top-center"/>
    )
}
