import type { Position } from '../types';
import { formatCurrency } from '../utils/format';
import { BrokerBadge } from '../components/BrokerBadge'; // <-- Import the badge

export function PortfolioView({ positions }: { positions: Position[] }) {
  return (
    <div className="space-y-12 animate-in fade-in duration-300">
      <div className="bg-white dark:bg-[#18181b] border border-zinc-200 dark:border-white/5 rounded-2xl overflow-hidden shadow-sm">
        <div className="p-6 border-b border-zinc-200 dark:border-white/5">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-white">Active Holdings</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="bg-zinc-50 dark:bg-white/[0.02] text-zinc-500 dark:text-zinc-400">
                <th className="font-medium py-3 px-6">Broker</th> {/* <-- New Header */}
                <th className="font-medium py-3 px-6">Symbol</th>
                <th className="font-medium py-3 px-6 text-right">Quantity</th>
                <th className="font-medium py-3 px-6 text-right">Market Value</th>
                <th className="font-medium py-3 px-6 text-right">Unrealized P/L</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-white/5">
              {positions.length === 0 ? (
                <tr><td colSpan={5} className="py-8 text-center text-zinc-500">No active positions found.</td></tr>
              ) : (
                positions.map((p) => (
                  <tr key={p.id} className="hover:bg-zinc-50 dark:hover:bg-white/[0.02]">
                    <td className="py-4 px-6"><BrokerBadge name={p.institution_name} /></td> {/* <-- New Data Cell */}
                    <td className="py-4 px-6 font-bold text-zinc-900 dark:text-white">{p.symbol}</td>
                    <td className="py-4 px-6 text-right text-zinc-600 dark:text-zinc-300">{p.quantity}</td>
                    <td className="py-4 px-6 text-right text-zinc-900 dark:text-white">{formatCurrency(p.market_value)}</td>
                    <td className={`py-4 px-6 text-right font-medium ${p.unrealized_pl >= 0 ? 'text-emerald-500' : 'text-rose-500'}`}>
                      {p.unrealized_pl >= 0 ? '+' : ''}{formatCurrency(p.unrealized_pl)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}