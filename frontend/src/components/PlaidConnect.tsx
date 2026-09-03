import { useCallback } from 'react';
import { usePlaidLink, type PlaidLinkOnSuccess } from 'react-plaid-link';
import { Wallet, Loader2 } from 'lucide-react';
import { useQueryClient, useQuery } from '@tanstack/react-query';

interface PlaidConnectProps {
  userId: string;
  jwt: string;
}

export function PlaidConnect({ userId, jwt }: PlaidConnectProps) {
  const queryClient = useQueryClient();

  // 1. Fetch Link Token using TanStack Query & secure JWT
  const { data: linkToken, isSuccess, isPending } = useQuery({
    queryKey: ['plaid-link-token', userId],
    queryFn: async () => {
      const res = await fetch('http://localhost:8080/api/v1/plaid/create-link-token', {
        method: 'POST',
        headers: { 
          'Authorization': `Bearer ${jwt}` 
        },
      });
      if (!res.ok) throw new Error('Failed to fetch link token');
      const data = await res.json();
      return data.link_token;
    },
    // We don't need to refetch the link token on window focus
    refetchOnWindowFocus: false,
    staleTime: Infinity, 
  });

  const onSuccess: PlaidLinkOnSuccess = useCallback(async (public_token, metadata) => {
    if (!public_token) {
      console.warn("Plaid Link succeeded, but no public token was returned.", metadata);
      return;
    }

    // Extract the dynamic institution details the user just selected
    const institutionId = metadata.institution?.institution_id || "";
    const institutionName = metadata.institution?.name || "Connected Broker";

    // 2. Securely exchange the token on the backend, using the JWT
    await fetch('http://localhost:8080/api/v1/plaid/exchange-public-token', {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${jwt}` 
      },
      body: JSON.stringify({ 
        public_token: public_token,
        institution_id: institutionId,
        institution_name: institutionName
      }),
    });
    
    // 3. MAGIC: Tell the global cache that the portfolio and transaction data is now stale.
    queryClient.invalidateQueries({ queryKey: ['portfolio', userId] });
    queryClient.invalidateQueries({ queryKey: ['transactions', userId] });
  }, [userId, jwt, queryClient]);

  const { open, ready } = usePlaidLink({
    token: linkToken || '',
    onSuccess,
  });

  return (
    <button
      onClick={() => open()}
      disabled={!ready || !isSuccess || isPending}
      className="flex items-center gap-2 bg-zinc-900 text-white hover:bg-zinc-800 dark:bg-white dark:text-black dark:hover:bg-zinc-200 px-5 py-2.5 rounded-xl font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-sm dark:shadow-[0_0_15px_rgba(255,255,255,0.1)]"
    >
      {isPending ? (
        <Loader2 size={18} className="animate-spin text-zinc-500 dark:text-zinc-400" />
      ) : (
        <Wallet size={18} />
      )}
      Connect Broker
    </button>
  );
}