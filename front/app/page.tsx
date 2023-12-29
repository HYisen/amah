import Image from 'next/image'
import Link from "next/link";


export default function Home() {
    return (
        <main>
            <p>Hello world.</p>
            <Image
                src="/next.svg"
                alt="Next.js Logo"
                width={180}
                height={37}
                priority
            />
            <Link href={"/config"}>Config</Link>
        </main>
    )
}
