export type Position = {
  id: string;
  symbol: string;
  quantity: number;
  market_value: number;
  unrealized_pl: number;
  broker?: string; // <-- Added
  institution_name?: string; // Match the JSON tag from Go
};

export type Transaction = {
  id: string;
  symbol: string;
  date: string;
  name: string;
  amount: number;
  type: string;
  broker?: string;
  datetime?: string; // <-- Added for high-resolution broker data
  institution_name?: string; // Match the JSON tag from Go
};