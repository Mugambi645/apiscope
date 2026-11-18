import { notFound } from "next/navigation";
import type { Metadata } from "next";
import Link from "next/link";
import { getApi, getPreview } from "@/lib/api";
import HealthBadge from "@/components/HealthBadge";
import Playground from "@/components/Playground";

export const dynamic = "force-dynamic";

type Props = { params: { slug: string } };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const api = await getApi(params.slug).catch(() => null);
  return api ? { title: `${api.name} · APIScope`, description: api.description } : { title: "Not found" };
}

export default async function ApiPage({ params }: Props) {
  const api = await getApi(params.slug);
  if (!api) notFound();
  const preview = await getPreview(api.slug);

  return (
    <>
      <p><Link href="/">← All APIs</Link></p>
      <h1>{api.name}</h1>
      <p className="lead">{api.description}</p>
      <div className="row">
        <HealthBadge health={api.health} />
        <span className="badge badge-muted">auth: {api.auth}</span>
        <span className="badge badge-muted">CORS {api.health.checks ? (api.health.cors_verified ? "verified" : "not seen") : api.cors ? "declared" : "no"}</span>
        {api.license && <span className="badge badge-muted">{api.license}</span>}
      </div>

      <a className="preview" href={api.docs_url} target="_blank" rel="noopener noreferrer">
        {preview?.favicon && /* eslint-disable-next-line @next/next/no-img-element */ <img src={preview.favicon} alt="" width={20} height={20} />}
        <div>
          <strong>{preview?.title || "Documentation"}</strong>
          <p>{preview?.description || api.docs_url}</p>
          <span className="muted">{new URL(api.docs_url).hostname}</span>
        </div>
      </a>

      <Playground api={api} />
    </>
  );
}
