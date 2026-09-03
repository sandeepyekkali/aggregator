import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layers, History, LogOut } from 'lucide-react';
import { PlaidConnect } from './components/PlaidConnect';
import { ThemeToggle } from './components/ThemeToggle';
import { PortfolioView } from './views/PortfolioView';
import { HistoryView } from './views/HistoryView';
import { AuthView } from './views/AuthView';
import { supabase } from './utils/supabase';
import { hasAccess } from './utils/tiers';
import type { Position, Transaction } from './types';
import type { Session } from '@supabase/supabase-js';

export default function App() {
  const [session, setSession] = useState<Session | null>(null);
  const [isLoadingAuth, setIsLoadingAuth] = useState(true);
  const [authEvent, setAuthEvent] = useState<string>(''); // <-- Track the auth event type
  const [activeTab, setActiveTab] = useState<'portfolio' | 'history'>('portfolio');

  useEffect(() => {
    supabase.auth.getSession().then(({ data: { session } }) => {
      setSession(session);
      setIsLoadingAuth(false);
    });

    const { data: { subscription } } = supabase.auth.onAuthStateChange((event, session) => {
      setSession(session);
      setAuthEvent(event); // <-- Capture the event (e.g., 'PASSWORD_RECOVERY')
      setIsLoadingAuth(false);
    });

    return () => subscription.unsubscribe();
  }, []);

  const fetchSecure = async (endpoint: string) => {
    if (!session?.access_token) throw new Error("No active session");
    
    const res = await fetch(`http://localhost:8080${endpoint}`, {
      headers: { 'Authorization': `Bearer ${session.access_token}` }
    });

    if (!res.ok) {
      if (res.status === 401) supabase.auth.signOut();
      throw new Error(`API Error: ${res.status}`);
    }
    return res.json();
  };

  const { data: rawPositions } = useQuery<Position[]>({
    queryKey: ['portfolio', session?.user?.id],
    queryFn: () => fetchSecure('/api/v1/portfolio'),
    enabled: !!session && authEvent !== 'PASSWORD_RECOVERY', 
    retry: 1
  });
  const positions = rawPositions || [];

  const { data: rawTransactions } = useQuery<Transaction[]>({
    queryKey: ['transactions', session?.user?.id],
    queryFn: () => fetchSecure('/api/v1/transactions'),
    enabled: !!session && authEvent !== 'PASSWORD_RECOVERY',
    retry: 1
  });
  const transactions = rawTransactions || [];

  if (isLoadingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-[#09090b]">
        <div className="w-6 h-6 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  // Intercept the flow if there's no session OR if the user is in recovery mode
  if (!session || authEvent === 'PASSWORD_RECOVERY') {
    return (
      <AuthView 
        isRecovering={authEvent === 'PASSWORD_RECOVERY'} 
        onRecovered={() => setAuthEvent('SIGNED_IN')} 
      />
    );
  }

  const userTier = session.user.app_metadata?.tier || 'basic';

  return (
    <div className="min-h-screen p-8 max-w-7xl mx-auto">
      <header className="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-8 gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-zinc-900 dark:text-white">Portfolio Dashboard</h1>
          <p className="text-zinc-500 dark:text-zinc-400 mt-1">Aggregated wealth and behavioral analytics</p>
        </div>
        
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-3 bg-white dark:bg-[#18181b] px-3 py-1.5 rounded-lg border border-zinc-200 dark:border-white/10 shadow-sm">
            <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {session.user.email}
            </span>
            
            <span className={`px-2 py-0.5 rounded text-xs font-semibold uppercase tracking-wider ${
              userTier === 'premium' ? 'bg-indigo-100 dark:bg-indigo-500/20 text-indigo-700 dark:text-indigo-400' :
              userTier === 'pro' ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400' :
              'bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400'
            }`}>
              {userTier}
            </span>

            <div className="w-px h-4 bg-zinc-200 dark:bg-white/10"></div>
            <button 
              onClick={() => supabase.auth.signOut()} 
              className="text-zinc-500 hover:text-rose-500 transition-colors"
              title="Sign Out"
            >
              <LogOut size={16} />
            </button>
          </div>

          <ThemeToggle />
          
          {hasAccess(userTier, 'basic') && (
            <PlaidConnect userId={session.user.id} jwt={session.access_token} />
          )}
        </div>
      </header>

      <div className="flex gap-4 mb-8 border-b border-zinc-200 dark:border-white/10 pb-4">
        <button 
          onClick={() => setActiveTab('portfolio')} 
          className={`flex items-center gap-2 font-medium transition-colors ${activeTab === 'portfolio' ? 'text-indigo-600 dark:text-indigo-400' : 'text-zinc-500 hover:text-zinc-900 dark:hover:text-white'}`}
        >
          <Layers size={18} /> Live Positions
        </button>
        <button 
          onClick={() => setActiveTab('history')} 
          className={`flex items-center gap-2 font-medium transition-colors ${activeTab === 'history' ? 'text-indigo-600 dark:text-indigo-400' : 'text-zinc-500 hover:text-zinc-900 dark:hover:text-white'}`}
        >
          <History size={18} /> Historical Analytics
        </button>
      </div>

      {activeTab === 'portfolio' ? (
        <PortfolioView positions={positions} />
      ) : (
        <HistoryView transactions={transactions} />
      )}
    </div>
  );
}