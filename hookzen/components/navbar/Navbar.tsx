import Link from "next/link"

function Navbar() {
  return (
    <div className='w-full max-w-7xl relative flex items-center justify-between px-7 py-4 md:px-4 md:py-4'>
      <Link href="/" aria-label="nav-logo" className="flex items-center gap-1 select-none">
        <img src={"/favicon.svg"} alt="nav-logo" className="w-6 h-6 object-contain" />
        <h1 className="text-xl font-bold text-white">Hookzen</h1>
      </Link>
    </div>
  )
}

export default Navbar