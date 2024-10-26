import React, { useState } from 'react';
import { TextField, Button, Box, Typography } from '@mui/material';
import axios from 'axios';

function UrlShortenerForm() {
  const [originalUrl, setOriginalUrl] = useState('');
  const [shortenedUrl, setShortenedUrl] = useState('');
  const [error, setError] = useState('');

  const validateUrl = (url) => {
    const urlPattern = /^https:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*)$/;
    return urlPattern.test(url);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setShortenedUrl('');

    if (!validateUrl(originalUrl)) {
      setError('Please enter a valid HTTPS URL.');
      return;
    }

    try {
      const response = await axios.post('http://localhost:8080/shorten', { url: originalUrl });
      setShortenedUrl(response.data.short_url);
    } catch (err) {
      setError('Error creating shortened URL. Please try again.');
      console.error(err);
    }
  };

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ maxWidth: 400, margin: 'auto', mt: 4 }}>
      <Typography variant="h4" component="h1" gutterBottom>
        URL Shortener
      </Typography>
      <TextField
        fullWidth
        label="Enter HTTPS URL to shorten"
        variant="outlined"
        value={originalUrl}
        onChange={(e) => setOriginalUrl(e.target.value)}
        margin="normal"
        required
        error={!!error}
        helperText={error}
      />
      <Button type="submit" variant="contained" color="primary" fullWidth sx={{ mt: 2 }}>
        Shorten URL
      </Button>
      {shortenedUrl && (
        <Typography variant="body1" sx={{ mt: 2 }}>
          Shortened URL: <a href={shortenedUrl} target="_blank" rel="noopener noreferrer">{shortenedUrl}</a>
        </Typography>
      )}
    </Box>
  );
}

export default UrlShortenerForm;
