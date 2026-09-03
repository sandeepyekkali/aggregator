import { useState } from 'react';

export function BrokerBadge({ name }: { name?: string }) {
  const [imgError, setImgError] = useState(false);
  
  const brokerName = name || 'Plaid'; 
  const normalized = brokerName.toUpperCase();

  // Map internal broker names to their official domains to fetch live logos
  const domainMap: Record<string, string> = {
    'PLAID': 'plaid.com',
    'IBKR': 'interactivebrokers.com',
    'INTERACTIVE BROKERS': 'interactivebrokers.com',
    'SNAP': 'snaptrade.com',
    'SNAPTRADE': 'snaptrade.com',
    'SCHWAB': 'schwab.com',
    'CHARLES SCHWAB': 'schwab.com',
    'FIDELITY': 'fidelity.com',
    'ROBINHOOD': 'robinhood.com',
  };

  // Scan the mapping to find the matching domain
  const domain = Object.keys(domainMap).reduce((acc, key) => {
    if (normalized.includes(key)) return domainMap[key];
    return acc;
  }, 'bank.com'); // generic fallback domain

  // Fetch the 64px square version of the logo
  const logoUrl = `https://logo.clearbit.com/${domain}?size=64`;

  // Fallback dot colors just in case the API blocks the request or the logo is missing
  let dotColor = 'bg-zinc-400';
  if (normalized.includes('PLAID')) dotColor = 'bg-emerald-500';
  if (normalized.includes('SNAP') || normalized.includes('IBKR')) dotColor = 'bg-rose-500';

  return (
    <span className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md bg-white dark:bg-zinc-900/50 text-zinc-700 dark:text-zinc-300 text-xs font-medium border border-zinc-200 dark:border-white/10 shadow-sm transition-all hover:bg-zinc-50 dark:hover:bg-zinc-800">
      {!imgError ? (
        <img 
          src={logoUrl} 
          alt={`${brokerName} logo`} 
          className="w-4 h-4 rounded-sm object-contain bg-white" // bg-white helps transparent PNGs pop in dark mode
          onError={() => setImgError(true)}
        />
      ) : (
        <span className={`w-1.5 h-1.5 rounded-full ${dotColor}`}></span>
      )}
      <span className="capitalize">{brokerName.toLowerCase()}</span>
    </span>
  );
}