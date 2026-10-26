import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import { TopicView } from "@/components/topic/topic-view";
import { getOrigin, getSite, serverGet } from "@/lib/server-api";
import type { TopicView as TopicViewData } from "@/lib/types";
import { htmlToText, summarize, topicUrl } from "@/lib/utils";
import { cache } from "react";

type Props = { params: Promise<{ parts: string[] }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

// /t/<slug>/<id>, /t/<slug>/<id>/<post number>, and the short /t/<id>
function parse(parts: string[]) {
  const decoded = parts.map((p) => decodeURIComponent(p));
  if (decoded.length === 1 && /^\d+$/.test(decoded[0])) {
    return { slug: "", id: Number(decoded[0]), postNumber: 0 };
  }
  const id = Number(decoded[1]);
  const postNumber = decoded[2] && /^\d+$/.test(decoded[2]) ? Number(decoded[2]) : 0;
  return { slug: decoded[0] ?? "", id: Number.isFinite(id) ? id : 0, postNumber };
}

const loadTopic = cache(async (id: number, postNumber: number, page: number) => {
  return serverGet<TopicViewData>("get-topic", { id, postNumber: postNumber || undefined, page: page || undefined });
});

function pageOf(searchParams: Record<string, string | string[] | undefined>): number {
  const value = Number(Array.isArray(searchParams.page) ? searchParams.page[0] : searchParams.page);
  return Number.isFinite(value) && value > 1 ? Math.floor(value) : 0;
}

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { id, postNumber } = parse((await params).parts);
  const page = pageOf(await searchParams);
  if (!id) {
    return {};
  }
  const res = await loadTopic(id, postNumber, page);
  if (res.status !== "ok") {
    return { robots: { index: false } };
  }
  const { topic, category, posts } = res.data;
  const site = await getSite();
  const origin = await getOrigin();
  const canonical = `${origin}${topicUrl(topic)}${page > 1 ? `?page=${page}` : ""}`;
  const description = summarize(topic.excerpt || htmlToText(posts[0]?.cooked ?? ""));
  const image = topic.imageUrl || site.settings.logo_url;
  const isMessage = topic.archetype === "private_message";
  return {
    title: { absolute: [topic.title, category?.name, site.settings.title || "Casnode"].filter(Boolean).join(" - ") },
    description,
    alternates: { canonical, types: { "application/rss+xml": `${origin}${topicUrl(topic)}.rss` } },
    robots: isMessage || !topic.visible || topic.deleted ? { index: false, follow: false } : undefined,
    openGraph: {
      type: "article",
      title: topic.title,
      description,
      url: canonical,
      images: image ? [image] : undefined,
      publishedTime: topic.createdAt,
      modifiedTime: topic.bumpedAt,
    },
    twitter: { card: topic.imageUrl ? "summary_large_image" : "summary", title: topic.title, description },
  };
}

export default async function Page({ params, searchParams }: Props) {
  const { slug, id, postNumber } = parse((await params).parts);
  const query = await searchParams;
  const page = pageOf(query);
  if (!id) {
    notFound();
  }
  const res = await loadTopic(id, postNumber, page);
  // a topic merged into another leads there
  const mergedInto = (res.data as unknown as { mergedInto?: { id: number; slug: string } } | null)?.mergedInto;
  if (res.status !== "ok" && mergedInto) {
    permanentRedirect(topicUrl(mergedInto));
  }
  if (res.status !== "ok") {
    notFound();
  }
  const view = res.data;
  if (slug !== view.topic.slug) {
    permanentRedirect(topicUrl(view.topic, postNumber));
  }

  const origin = await getOrigin();
  const first = view.posts.find((p) => p.postNumber === 1);
  const author = view.users[view.topic.userId];
  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "DiscussionForumPosting",
    headline: view.topic.title,
    url: `${origin}${topicUrl(view.topic)}`,
    datePublished: view.topic.createdAt,
    dateModified: view.topic.bumpedAt,
    text: first ? htmlToText(first.cooked).slice(0, 1000) : undefined,
    author: author ? { "@type": "Person", name: author.username, url: `${origin}/u/${encodeURIComponent(author.username)}` } : undefined,
    interactionStatistic: [
      { "@type": "InteractionCounter", interactionType: "https://schema.org/LikeAction", userInteractionCount: view.topic.likeCount },
      { "@type": "InteractionCounter", interactionType: "https://schema.org/CommentAction", userInteractionCount: view.topic.replyCount },
      { "@type": "InteractionCounter", interactionType: "https://schema.org/ViewAction", userInteractionCount: view.topic.views },
    ],
    comment: view.posts
      .filter((p) => p.postNumber > 1 && p.postType === 1 && !p.deleted)
      .map((p) => ({
        "@type": "Comment",
        text: htmlToText(p.cooked).slice(0, 500),
        datePublished: p.createdAt,
        author: view.users[p.userId] ? { "@type": "Person", name: view.users[p.userId].username } : undefined,
        url: `${origin}${topicUrl(view.topic, p.postNumber)}`,
      })),
  };
  const pages = Math.ceil(view.stream.length / 20);
  if (page > pages) {
    notFound();
  }

  return (
    <>
      {view.topic.archetype !== "private_message" ? (
        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd).replace(/</g, "\\u003c") }} />
      ) : null}
      <TopicView initial={view} targetPostNumber={postNumber} />
      {pages > 1 ? (
        <nav className="sr-only" aria-label="pagination">
          {page > 1 ? (
            <a href={`${topicUrl(view.topic)}${page > 2 ? `?page=${page - 1}` : ""}`} rel="prev">
              prev
            </a>
          ) : null}
          {(page || 1) < pages ? (
            <a href={`${topicUrl(view.topic)}?page=${(page || 1) + 1}`} rel="next">
              next
            </a>
          ) : null}
        </nav>
      ) : null}
    </>
  );
}
