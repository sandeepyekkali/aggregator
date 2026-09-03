export const TIER_WEIGHTS: Record<string, number> = {
  basic: 10,
  pro: 20,
  premium: 30,
  elite: 40,
};

/**
 * Checks if the user's current tier meets or exceeds the required tier.
 */
export const hasAccess = (userTier: string, requiredTier: string): boolean => {
  const userWeight = TIER_WEIGHTS[userTier] || TIER_WEIGHTS['basic'];
  const requiredWeight = TIER_WEIGHTS[requiredTier] || 999;
  
  return userWeight >= requiredWeight;
};