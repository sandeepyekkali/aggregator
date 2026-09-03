import { useMemo } from 'react';
import type { Transaction } from '../types';

export function useBehavioralAnalytics(transactions: Transaction[]) {
  return useMemo(() => {
    if (transactions.length === 0) return null;

    let totalTradeVolume = 0;
    let tradeCount = 0;
    const assetVolumes: Record<string, number> = {};
    const monthlyActivity: Record<string, { month: string; buy: number; sell: number }> = {};

    transactions.forEach(t => {
      const isTrade = t.type === 'buy' || t.type === 'sell';
      const absAmount = Math.abs(t.amount);
      const symbol = t.symbol || 'Unknown';

      if (isTrade) {
        totalTradeVolume += absAmount;
        tradeCount++;
        if (symbol !== 'CASH') {
          assetVolumes[symbol] = (assetVolumes[symbol] || 0) + absAmount;
        }
      }

      const dateObj = new Date(t.date);
      const monthKey = `${dateObj.getFullYear()}-${String(dateObj.getMonth() + 1).padStart(2, '0')}`;
      const displayMonth = dateObj.toLocaleDateString(undefined, { month: 'short', year: '2-digit' });

      if (!monthlyActivity[monthKey]) {
        monthlyActivity[monthKey] = { month: displayMonth, buy: 0, sell: 0 };
      }
      
      if (t.type === 'buy') monthlyActivity[monthKey].buy += absAmount;
      if (t.type === 'sell') monthlyActivity[monthKey].sell += absAmount;
    });

    const topAssets = Object.keys(assetVolumes)
      .map(key => ({ symbol: key, volume: assetVolumes[key] }))
      .sort((a, b) => b.volume - a.volume)
      .slice(0, 5);

    const activityTrend = Object.keys(monthlyActivity)
      .sort()
      .map(key => monthlyActivity[key]);

    return {
      totalTransactions: transactions.length,
      tradeCount,
      totalTradeVolume,
      avgTradeSize: tradeCount > 0 ? totalTradeVolume / tradeCount : 0,
      topAssets,
      activityTrend
    };
  }, [transactions]);
}