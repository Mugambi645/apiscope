import SearchClient from "@/components/SearchClient";

export default function Home() {
  return (
    <>
      <h1>Find an open API, then actually run it.</h1>
      <p className="lead">Every listing is probed on a schedule, so the badge shows what is working right now.</p>
      <SearchClient />
    </>
  );
}
