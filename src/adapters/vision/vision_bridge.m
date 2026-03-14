#import <Vision/Vision.h>
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

// visionRecognizeText performs OCR on PNG data using Apple's Vision framework.
// Input: PNG-encoded image bytes and a BCP 47 language code (e.g., "en-US", "zh-Hans").
// Output: A malloc'd UTF-8 C string containing recognized text lines joined by newlines.
// Caller must free the returned pointer.
char* visionRecognizeText(const unsigned char* pngData, size_t length, const char* language) {
	@autoreleasepool {
		// Decode PNG bytes to an NSImage
		NSData* imageData = [NSData dataWithBytes:pngData length:length];
		NSImage* nsImage = [[NSImage alloc] initWithData:imageData];
		if (!nsImage || nsImage.size.width == 0 || nsImage.size.height == 0) {
			return strdup("");
		}

		// Convert NSImage to CGImage
		CGImageRef cgImage = [nsImage CGImageForProposedRect:NULL context:nil hints:nil];
		if (!cgImage) {
			return strdup("");
		}

		// Create the text recognition request
		VNRecognizeTextRequest* request = [[VNRecognizeTextRequest alloc] init];
		request.recognitionLevel = VNRequestTextRecognitionLevelAccurate;

		// Set language if provided
		if (language && strlen(language) > 0) {
			@try {
				NSString* langString = [NSString stringWithUTF8String:language];
				request.recognitionLanguages = @[langString];
			} @catch (NSException* exception) {
				// Invalid language code; proceed with system default
			}
		}

		// Perform the request
		NSError* error = nil;
		VNImageRequestHandler* handler = [[VNImageRequestHandler alloc]
			initWithCGImage:cgImage options:@{}];
		[handler performRequests:@[request] error:&error];

		// Collect results
		NSMutableString* result = [NSMutableString string];
		for (VNRecognizedTextObservation* observation in request.results) {
			VNRecognizedText* recognizedText = [[observation topCandidates:1] firstObject];
			if (recognizedText && recognizedText.string.length > 0) {
				[result appendString:recognizedText.string];
				[result appendString:@"\n"];
			}
		}

		// Convert to C string and return (caller must free)
		const char* resultUTF8 = [result UTF8String] ?: "";
		return strdup(resultUTF8);
	}
}