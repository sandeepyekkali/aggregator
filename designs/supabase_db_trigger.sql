-- 1. Create the function that injects the tier
CREATE OR REPLACE FUNCTION public.handle_new_user_tier()
RETURNS TRIGGER AS $$
BEGIN
  -- Safely merge '{"tier": "basic"}' into their app_metadata
  NEW.raw_app_meta_data := coalesce(NEW.raw_app_meta_data, '{}'::jsonb) || '{"tier": "basic"}'::jsonb;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 2. Attach the function to the auth.users table
DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
  BEFORE INSERT ON auth.users
  FOR EACH ROW EXECUTE PROCEDURE public.handle_new_user_tier();