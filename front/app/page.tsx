import Image from 'next/image'

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
        </main>
    )
}
