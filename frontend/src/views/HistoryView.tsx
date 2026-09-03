import { useState, useMemo } from 'react';
import { History, Activity, TrendingUp, Wallet, ArrowLeft, ArrowRight } from 'lucide-react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Legend, Cell } from 'recharts';
import type { Transaction } from '../types';
import { formatCurrency } from '../utils/format';
import { useBehavioralAnalytics } from '../hooks/useBehavioralAnalytics';
import { BrokerBadge } from '../components/BrokerBadge';

export function HistoryView({ transactions }: { transactions: Transaction[] }) {
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;
  
  const analytics = useBehavioralAnalytics(transactions);

  const totalPages = Math.ceil(transactions.length / pageSize);
  const paginatedTransactions = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return transactions.slice(start, start + pageSize);
  }, [transactions, currentPage]);

  if (!analytics) {
    return <div className="text-zinc-500 text-center py-8">No transaction history found.</div>;
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      {/* KPI Dashboard */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <div className="text-zinc-500 dark:text-zinc-400 text-sm font-medium mb-2 flex items-center gap-2"><History size={16}/> Total Events</div>
          <div className="text-3xl font-bold text-zinc-900 dark:text-white">{analytics.totalTransactions}</div>
        </div>
        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <div className="text-zinc-500 dark:text-zinc-400 text-sm font-medium mb-2 flex items-center gap-2"><Activity size={16}/> Lifetime Volume</div>
          <div className="text-3xl font-bold text-zinc-900 dark:text-white">{formatCurrency(analytics.totalTradeVolume)}</div>
        </div>
        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <div className="text-zinc-500 dark:text-zinc-400 text-sm font-medium mb-2 flex items-center gap-2"><Wallet size={16}/> Avg. Conviction</div>
          <div className="text-3xl font-bold text-indigo-600 dark:text-indigo-400">{formatCurrency(analytics.avgTradeSize)}</div>
        </div>
        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <div className="text-zinc-500 dark:text-zinc-400 text-sm font-medium mb-2 flex items-center gap-2"><TrendingUp size={16}/> Top Asset</div>
          <div className="text-3xl font-bold text-zinc-900 dark:text-white">{analytics.topAssets[0]?.symbol || 'N/A'}</div>
        </div>
      </div>

      {/* Behavioral Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-6">Directional Bias (Buy vs Sell)</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={analytics.activityTrend}>
                <XAxis dataKey="month" stroke="#71717a" fontSize={12} tickLine={false} axisLine={false} />
                <YAxis stroke="#71717a" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => `$${val/1000}k`} />
                <Tooltip formatter={(value: any) => formatCurrency(Number(value))} contentStyle={{ backgroundColor: '#18181b', borderColor: '#3f3f46', color: '#fff' }} />
                <Legend iconType="circle" />
                <Bar dataKey="buy" name="Capital Deployed (Buy)" stackId="a" fill="#10b981" radius={[0, 0, 4, 4]} />
                <Bar dataKey="sell" name="Capital Recovered (Sell)" stackId="a" fill="#f43f5e" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-white dark:bg-[#18181b] p-6 rounded-2xl border border-zinc-200 dark:border-white/5 shadow-sm">
          <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-6">Concentration Bias (Top 5)</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={analytics.topAssets} layout="vertical" margin={{ left: 20 }}>
                <XAxis type="number" hide />
                <YAxis dataKey="symbol" type="category" stroke="#71717a" fontSize={12} tickLine={false} axisLine={false} />
                <Tooltip formatter={(value: any) => formatCurrency(Number(value))} cursor={{fill: 'transparent'}} contentStyle={{ backgroundColor: '#18181b', borderColor: '#3f3f46', color: '#fff' }} />
                <Bar dataKey="volume" name="Traded Volume" radius={[0, 4, 4, 0]}>
                  {analytics.topAssets.map((_, i) => (
                    <Cell key={`cell-${i}`} fill={i === 0 ? '#6366f1' : '#4f46e580'} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Paginated Table */}
      <div className="bg-white dark:bg-[#18181b] border border-zinc-200 dark:border-white/5 rounded-2xl overflow-hidden shadow-sm flex flex-col">
        <div className="p-6 border-b border-zinc-200 dark:border-white/5 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">Chronological Ledger</h3>
          <span className="text-sm text-zinc-500 dark:text-zinc-400">Showing {paginatedTransactions.length} of {transactions.length} events</span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="bg-zinc-50 dark:bg-white/[0.02] text-zinc-500 dark:text-zinc-400">
                <th className="font-medium py-3 px-6">Broker</th>
                <th className="font-medium py-3 px-6">Date</th>
                <th className="font-medium py-3 px-6">Type</th>
                <th className="font-medium py-3 px-6">Asset</th>
                <th className="font-medium py-3 px-6 text-right">Amount</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-white/5">
              {paginatedTransactions.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-8 text-center text-zinc-500">No transactions recorded.</td>
                </tr>
              ) : (
                paginatedTransactions.map((t) => (
                  <tr key={t.id} className="hover:bg-zinc-50 dark:hover:bg-white/[0.02]">
                    <td className="py-4 px-6">
                      <BrokerBadge name={t.institution_name} />
                    </td>
                    <td className="py-4 px-6 text-zinc-600 dark:text-zinc-300">
                      {t.datetime ? (
                        <div className="flex flex-col">
                          <span className="font-medium text-zinc-900 dark:text-white">
                            {new Date(t.datetime).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                          </span>
                          <span className="text-xs text-zinc-500">
                            {new Date(t.datetime).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
                          </span>
                        </div>
                      ) : (
                        new Date(t.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
                      )}
                    </td>
                    <td className="py-4 px-6">
                      <span className={`px-2.5 py-1 rounded-full text-xs font-medium uppercase tracking-wider ${
                        t.type === 'buy' ? 'bg-emerald-100 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400' : 
                        t.type === 'sell' ? 'bg-rose-100 dark:bg-rose-500/10 text-rose-700 dark:text-rose-400' : 
                        'bg-zinc-100 dark:bg-white/10 text-zinc-700 dark:text-zinc-300'
                      }`}>{t.type}</span>
                    </td>
                    <td className="py-4 px-6 font-medium text-zinc-900 dark:text-white">{t.name}</td>
                    <td className="py-4 px-6 text-right font-medium text-zinc-900 dark:text-white">{formatCurrency(Math.abs(t.amount))}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        {totalPages > 1 && (
          <div className="p-4 border-t border-zinc-200 dark:border-white/5 flex items-center justify-between bg-zinc-50/50 dark:bg-white/[0.01]">
            <button 
              onClick={() => setCurrentPage(p => Math.max(1, p - 1))} 
              disabled={currentPage === 1} 
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-zinc-700 dark:text-zinc-300 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-white/10 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50 dark:hover:bg-zinc-700 transition-colors"
            >
              <ArrowLeft size={16} /> Previous
            </button>
            <span className="text-sm text-zinc-500 dark:text-zinc-400">
              Page <span className="font-semibold text-zinc-900 dark:text-white">{currentPage}</span> of {totalPages}
            </span>
            <button 
              onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))} 
              disabled={currentPage === totalPages} 
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-zinc-700 dark:text-zinc-300 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-white/10 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50 dark:hover:bg-zinc-700 transition-colors"
            >
              Next <ArrowRight size={16} />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}