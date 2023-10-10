type ExternalLinkProps = {
  href: string;
  className?: string;
  children?: React.ReactNode;
};

export const ExternalLink = ({ href, className, children }: ExternalLinkProps) => (
  <a
    rel="noopener noreferrer"
    target="_blank"
    href={href}
    className={className}
  >
    {children}
  </a>
);

export const DocsLink = ({ href }: { href: string; }) => (
  <ExternalLink href={href} className="text-blue-400 visited:text-blue-400">{href}</ExternalLink>
);
