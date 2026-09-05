import type { ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

import worklog from "../../../01_AI_WORKLOG_CURRENT.md?raw";
import styles from "./AIWorklogPage.module.css";

interface HeadingEntry {
  level: number;
  text: string;
  id: string;
}

function slugify(value: string): string {
  return value
    .toLocaleLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}]+/gu, "-")
    .replace(/^-|-$/g, "");
}

function headings(source: string): readonly HeadingEntry[] {
  const used = new Map<string, number>();
  return source.split("\n").flatMap((line) => {
    const match = /^(#{1,3})\s+(.+?)\s*$/.exec(line);
    if (!match) return [];
    const base = slugify(match[2].replace(/[`*_]/g, "")) || "section";
    const occurrence = (used.get(base) ?? 0) + 1;
    used.set(base, occurrence);
    return [
      {
        level: match[1].length,
        text: match[2].replace(/[`*_]/g, ""),
        id: occurrence === 1 ? base : `${base}-${occurrence}`,
      },
    ];
  });
}

function markdownComponents(source: string): Components {
  const expected = [...headings(source)];
  const heading = (level: number) =>
    function WorklogHeading({ children }: { children?: ReactNode }) {
      const entry = expected.find((candidate) => candidate.level === level);
      if (entry) expected.splice(expected.indexOf(entry), 1);
      const Tag = `h${level}` as "h1" | "h2" | "h3";
      return <Tag id={entry?.id}>{children}</Tag>;
    };
  return {
    h1: heading(1),
    h2: heading(2),
    h3: heading(3),
    table: ({ children, ...props }) => (
      <div className={styles.tableScroll} data-table-scroll tabIndex={0}>
        <table {...props}>{children}</table>
      </div>
    ),
  };
}

export function WorklogMarkdown({ source }: { source: string }) {
  return (
    <ReactMarkdown
      components={markdownComponents(source)}
      remarkPlugins={[remarkGfm]}
    >
      {source}
    </ReactMarkdown>
  );
}

export function AIWorklogPage() {
  const tableOfContents = headings(worklog).filter((entry) => entry.level > 1);
  return (
    <section className={styles.page}>
      <nav aria-label="On this page" className={styles.toc}>
        <strong>On this page</strong>
        <ol>
          {tableOfContents.map((entry) => (
            <li className={styles[`level${entry.level}`]} key={entry.id}>
              <a href={`#${entry.id}`}>{entry.text}</a>
            </li>
          ))}
        </ol>
      </nav>
      <article className={styles.article}>
        <WorklogMarkdown source={worklog} />
      </article>
    </section>
  );
}
