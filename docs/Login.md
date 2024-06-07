# Logging in/Authentication

## Resources
https://www.cloudflare.com/learning/ssl/what-is-asymmetric-encryption/

Ms. Ramirez is also a great source for knowledge on this topic - she's the one who taught me! (Take cybersecurity, it's probably the best course for anyone looking to get into tech.)

## Steps
1. The frontend requests the RSA public key from the server
2. The frontend encrypts its plaintext password with the public key and sends it to the backend
3. The backend decrypts that result with its private key
4. The backend passes the plaintext result through the bcrypt hashing algorithm, if they match it will either look for or generate a new certificate for that user, returning its credentials.